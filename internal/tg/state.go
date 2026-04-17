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
	case store.FileTypeUnit:
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

	case store.FileTypeUnknown:
		diags = []protocol.Diagnostic{}
	}

	s.Configs[filename] = st

	return diags
}

func (s *State) Hover(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.HoverResponse {
	st, ok := s.Configs[docURI.Filename()]
	if !ok {
		return newEmptyHoverResponse(id)
	}

	l.Debug(
		"Hovering over character",
		"uri", docURI,
		"position", position,
		"fileType", st.FileType,
	)

	switch st.FileType {
	case store.FileTypeUnit:
		return s.hoverUnit(l, id, st, position)
	case store.FileTypeStack:
		return s.hoverStack(l, id, st, position)
	case store.FileTypeValues:
		return s.hoverValues(l, id, st, position)
	case store.FileTypeUnknown:
		return newEmptyHoverResponse(id)
	}

	return newEmptyHoverResponse(id)
}

func (s *State) hoverUnit(l logger.Logger, id int, st store.Store, position protocol.Position) lsp.HoverResponse {
	l.Debug("Config", "config", st.Cfg)

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

func (s *State) hoverStack(l logger.Logger, id int, st store.Store, position protocol.Position) lsp.HoverResponse {
	target, context := hover.GetStackHoverTargetWithContext(l, st, position)

	l.Debug(
		"Stack hover with context",
		"target", target,
		"context", context,
	)

	if target == "" {
		return newEmptyHoverResponse(id)
	}

	switch context {
	case hover.HoverContextStackUnit:
		return newStackUnitHoverResponse(id, target)
	case hover.HoverContextStackSource:
		return newStackSourceHoverResponse(id, target)
	case hover.HoverContextStackPath:
		return newStackPathHoverResponse(id, target)
	case hover.HoverContextStackBlock:
		return newStackBlockHoverResponse(id, target)
	}

	return newEmptyHoverResponse(id)
}

func newStackUnitHoverResponse(id int, unitName string) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
		Result: lsp.HoverResult{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**Unit: `" + unitName + "`**\n\nA unit block defines a single infrastructure component in a Terragrunt stack.\n\nEach unit has a source (where the Terraform code lives) and a path (where it will be deployed).",
			},
		},
	}
}

func newStackSourceHoverResponse(id int, source string) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
		Result: lsp.HoverResult{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**Source: `" + source + "`**\n\nThe source attribute specifies where the Terraform module or configuration is located.\n\nThis can be a local path, Git repository, or other supported Terraform module sources.",
			},
		},
	}
}

func newStackPathHoverResponse(id int, path string) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
		Result: lsp.HoverResult{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**Path: `" + path + "`**\n\nThe path attribute specifies the relative directory where this unit will be deployed.\n\nThis path is relative to the stack directory and determines where Terragrunt will run commands for this unit.",
			},
		},
	}
}

func newStackBlockHoverResponse(id int, stackName string) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
		Result: lsp.HoverResult{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**Stack: `" + stackName + "`**\n\nA stack block defines a nested stack within the current stack.\n\nNested stacks allow you to organize and compose multiple related infrastructure units together.",
			},
		},
	}
}

func (s *State) hoverValues(l logger.Logger, id int, st store.Store, position protocol.Position) lsp.HoverResponse {
	target, context := hover.GetValuesHoverTargetWithContext(l, st, position)

	l.Debug(
		"Values hover with context",
		"target", target,
		"context", context,
	)

	if target == "" {
		return newEmptyHoverResponse(id)
	}

	switch context {
	case hover.HoverContextValuesVariable:
		return newValuesVariableHoverResponse(id, target)
	case hover.HoverContextValuesDependency:
		return newValuesDependencyHoverResponse(id, target)
	}

	return newEmptyHoverResponse(id)
}

func newValuesVariableHoverResponse(id int, variable string) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
		Result: lsp.HoverResult{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**Variable: `" + variable + "`**\n\nThis appears to be a variable defined in the values block.\n\nValues files are used to define dynamic input values for units in Terragrunt stacks.",
			},
		},
	}
}

func newValuesDependencyHoverResponse(id int, dependency string) lsp.HoverResponse {
	return lsp.HoverResponse{
		Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
		Result: lsp.HoverResult{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**Dependency: `" + dependency + "`**\n\nThis is a reference to a dependency unit defined elsewhere in your Terragrunt configuration.\n\nThe dependency must be declared before it can be referenced in a values file.",
			},
		},
	}
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
	st, ok := s.Configs[docURI.Filename()]
	if !ok {
		return newEmptyDefinitionResponse(id, docURI, position)
	}

	l.Debug(
		"Definition requested",
		"uri", docURI,
		"position", position,
		"fileType", st.FileType,
	)

	switch st.FileType {
	case store.FileTypeUnit:
		return s.definitionUnit(l, id, st, docURI, position)
	case store.FileTypeStack:
		return s.definitionStack(l, id, st, docURI, position)
	case store.FileTypeValues:
		return s.definitionValues(l, id, st, docURI, position)
	case store.FileTypeUnknown:
		return newEmptyDefinitionResponse(id, docURI, position)
	}

	return newEmptyDefinitionResponse(id, docURI, position)
}

func (s *State) definitionUnit(l logger.Logger, id int, st store.Store, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
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

func (s *State) definitionStack(l logger.Logger, id int, st store.Store, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	currentDir := filepath.Dir(docURI.Filename())

	target, context := definition.GetStackDefinitionTargetWithContext(l, st, position, currentDir)

	l.Debug(
		"Stack definition discovered",
		"target", target,
		"context", context,
	)

	if target == "" {
		return newEmptyDefinitionResponse(id, docURI, position)
	}

	switch context {
	case definition.DefinitionContextUnitSource:
		if resolved := definition.ResolveUnitSourceLocation(target, currentDir); resolved != "" {
			return newStackDefinitionResponse(id, resolved)
		}

		l.Debug("Could not resolve unit source location", "source", target)
	case definition.DefinitionContextStackSource:
		if resolved := definition.ResolveStackSourceLocation(target, currentDir); resolved != "" {
			return newStackDefinitionResponse(id, resolved)
		}

		l.Debug("Could not resolve stack source location", "source", target)
	case definition.DefinitionContextStackPath:
		return newStackDefinitionResponse(id, target)
	}

	return newEmptyDefinitionResponse(id, docURI, position)
}

func (s *State) definitionValues(l logger.Logger, id int, st store.Store, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	target, context := definition.GetValuesDefinitionTargetWithContext(l, st, position)

	l.Debug(
		"Values definition discovered",
		"target", target,
		"context", context,
	)

	if target == "" {
		return newEmptyDefinitionResponse(id, docURI, position)
	}

	//nolint:gocritic
	switch context {
	case definition.DefinitionContextValuesDependency:
		if resolved, ok := definition.ResolveValuesDependencyPath(target, docURI.Filename()); ok {
			return newStackDefinitionResponse(id, resolved)
		}

		l.Debug("Could not resolve values dependency path", "dependency", target)
	}

	return newEmptyDefinitionResponse(id, docURI, position)
}

func newStackDefinitionResponse(id int, resolved string) lsp.DefinitionResponse {
	return lsp.DefinitionResponse{
		Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
		Result: protocol.Location{
			URI: uri.File(resolved),
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
		},
	}
}

func (s *State) TextDocumentCompletion(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.CompletionResponse {
	st, ok := s.Configs[docURI.Filename()]
	if !ok {
		return lsp.CompletionResponse{
			Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
			Result:   []protocol.CompletionItem{},
		}
	}

	items := completion.GetCompletions(l, st, position)

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
	st, ok := s.Configs[docURI.Filename()]
	if !ok {
		return lsp.FormatResponse{
			Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
			Result:   []protocol.TextEdit{},
		}
	}

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
