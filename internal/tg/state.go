package tg

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/lsp"
	"terragrunt-ls/internal/tg/completion"
	"terragrunt-ls/internal/tg/definition"
	"terragrunt-ls/internal/tg/hover"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"

	"github.com/gruntwork-io/terragrunt/pkg/config"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type State struct {
	// Map of file names to Terragrunt configs
	Configs map[string]store.Store
}

func NewState() State {
	return State{Configs: map[string]store.Store{}}
}

func (s *State) OpenDocument(ctx context.Context, l logger.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	l.Debug(
		"Opening document",
		"uri", docURI,
		"text", text,
	)

	return s.updateState(ctx, l, docURI, text)
}

func (s *State) UpdateDocument(ctx context.Context, l logger.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	l.Debug(
		"Updating document",
		"uri", docURI,
		"text", text,
	)

	return s.updateState(ctx, l, docURI, text)
}

func (s *State) updateState(ctx context.Context, l logger.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	filename := docURI.Filename()
	fileType := DetectFileType(filename)

	// Ignore errors from AST indexing since we'll get the same errors from the Terragrunt parser just below
	indexedAST, _ := ast.ParseHCLFile(filename, []byte(text))

	st := store.Store{
		AST:      indexedAST,
		CfgAsCty: cty.NilVal,
		Document: text,
		FileType: fileType,
	}

	var diags []protocol.Diagnostic

	switch fileType {
	case store.FileTypeTerragrunt:
		cfg, unitDiags := ParseTerragruntBuffer(ctx, l, filename, text)

		l.Debug(
			"Config",
			"uri", docURI,
			"config", cfg,
		)

		cfgAsCty := cty.NilVal

		if cfg != nil {
			if converted, err := config.TerragruntConfigAsCty(cfg); err == nil {
				cfgAsCty = converted
			}
		}

		st.Cfg = cfg
		st.CfgAsCty = cfgAsCty
		diags = unitDiags

	case store.FileTypeStack:
		stackCfg, stackDiags := ParseStackBuffer(ctx, l, filename, text)

		l.Debug(
			"Stack Config",
			"uri", docURI,
			"config", stackCfg,
		)

		st.StackCfg = stackCfg
		diags = stackDiags

	case store.FileTypeValues:
		// Values files are generated; only store the document for formatting.
		diags = []protocol.Diagnostic{}
	}

	s.Configs[filename] = st

	return diags
}

func (s *State) Hover(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.HoverResponse {
	st := s.Configs[docURI.Filename()]

	l.Debug(
		"Hovering over character",
		"uri", docURI,
		"position", position,
	)

	if st.FileType != store.FileTypeTerragrunt {
		return newEmptyHoverResponse(id)
	}

	l.Debug(
		"Config",
		"uri", docURI,
		"config", st.Cfg,
	)

	word, context := hover.GetHoverTargetWithContext(l, st, position)

	l.Debug(
		"Hovering with context",
		"word", word,
		"context", context,
	)

	if word == "" {
		return newEmptyHoverResponse(id)
	}

	//nolint:gocritic
	switch context {
	case hover.HoverContextLocal:
		if st.Cfg == nil {
			return newEmptyHoverResponse(id)
		}

		if _, ok := st.Cfg.Locals[word]; !ok {
			return newEmptyHoverResponse(id)
		}

		if st.CfgAsCty.IsNull() {
			return newEmptyHoverResponse(id)
		}

		locals := st.CfgAsCty.GetAttr("locals")
		localVal := locals.GetAttr(word)

		f := hclwrite.NewEmptyFile()
		rootBody := f.Body()
		rootBody.SetAttributeValue(word, localVal)

		return lsp.HoverResponse{
			Response: lsp.Response{
				RPC: lsp.RPCVersion,
				ID:  &id,
			},
			Result: lsp.HoverResult{
				Contents: protocol.MarkupContent{
					Kind:  protocol.Markdown,
					Value: text.WrapAsHCLCodeFence(strings.TrimSpace(string(f.Bytes()))),
				},
			},
		}
	}

	return newEmptyHoverResponse(id)
}

func newEmptyHoverResponse(id int) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
	}
}

func (s *State) Definition(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	st := s.Configs[docURI.Filename()]

	l.Debug(
		"Definition requested",
		"uri", docURI,
		"position", position,
	)

	if st.FileType != store.FileTypeTerragrunt {
		return newEmptyDefinitionResponse(id, docURI, position)
	}

	target, context := definition.GetDefinitionTargetWithContext(l, st, position)

	l.Debug(
		"Definition discovered",
		"target", target,
		"context", context,
	)

	if target == "" {
		return newEmptyDefinitionResponse(id, docURI, position)
	}

	//nolint:gocritic
	switch context {
	case definition.DefinitionContextInclude:
		l.Debug(
			"Store content",
			"store", st,
		)

		if st.Cfg == nil {
			return newEmptyDefinitionResponse(id, docURI, position)
		}

		l.Debug(
			"Includes",
			"includes", st.Cfg.ProcessedIncludes,
		)

		for _, include := range st.Cfg.ProcessedIncludes {
			if include.Name == target {
				l.Debug(
					"Jumping to target",
					"include", include,
				)

				defURI := uri.File(include.Path)

				l.Debug(
					"URI of target",
					"URI", defURI,
				)

				return lsp.DefinitionResponse{
					Response: lsp.Response{
						RPC: lsp.RPCVersion,
						ID:  &id,
					},
					Result: protocol.Location{
						URI: defURI,
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      0,
								Character: 0,
							},
							End: protocol.Position{
								Line:      0,
								Character: 0,
							},
						},
					},
				}
			}
		}
	case definition.DefinitionContextDependency:
		l.Debug(
			"Store content",
			"store", st,
		)

		if st.Cfg == nil {
			return newEmptyDefinitionResponse(id, docURI, position)
		}

		l.Debug(
			"Dependencies",
			"dependencies", st.Cfg.TerragruntDependencies,
		)

		for _, dep := range st.Cfg.TerragruntDependencies {
			if dep.Name == target {
				l.Debug(
					"Jumping to target",
					"dependency", dep,
				)

				path := dep.ConfigPath.AsString()

				defURI := uri.File(path)
				if !filepath.IsAbs(path) {
					defURI = uri.File(filepath.Join(filepath.Dir(docURI.Filename()), path, "terragrunt.hcl"))
				}

				_, err := os.Stat(defURI.Filename())
				if err != nil {
					l.Warn(
						"Dependency does not exist",
						"dependency", dep,
						"error", err,
					)

					return newEmptyDefinitionResponse(id, docURI, position)
				}

				l.Debug(
					"URI of target",
					"URI", defURI,
				)

				return lsp.DefinitionResponse{
					Response: lsp.Response{
						RPC: lsp.RPCVersion,
						ID:  &id,
					},
					Result: protocol.Location{
						URI: defURI,
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      0,
								Character: 0,
							},
							End: protocol.Position{
								Line:      0,
								Character: 0,
							},
						},
					},
				}
			}
		}
	}

	return newEmptyDefinitionResponse(id, docURI, position)
}

func newEmptyDefinitionResponse(id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	return lsp.DefinitionResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
		Result: protocol.Location{
			URI: docURI,
			Range: protocol.Range{
				Start: position,
				End:   position,
			},
		},
	}
}

func (s *State) TextDocumentCompletion(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.CompletionResponse {
	items := completion.GetCompletions(l, s.Configs[docURI.Filename()], position)

	response := lsp.CompletionResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: items,
	}

	return response
}

func (s *State) TextDocumentFormatting(l logger.Logger, id int, docURI protocol.DocumentURI) lsp.FormatResponse {
	st := s.Configs[docURI.Filename()]

	l.Debug(
		"Formatting requested",
		"uri", docURI,
	)

	formatted := hclwrite.Format([]byte(st.Document))

	return lsp.FormatResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
		Result: []protocol.TextEdit{
			{
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      0,
						Character: 0,
					},
					End: getEndOfDocument(st.Document),
				},
				NewText: string(formatted),
			},
		},
	}
}

func getEndOfDocument(doc string) protocol.Position {
	lines := strings.Split(doc, "\n")

	return protocol.Position{
		Line:      uint32(len(lines) - 1),
		Character: uint32(len(lines[len(lines)-1])),
	}
}
