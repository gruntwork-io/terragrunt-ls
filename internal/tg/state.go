package tg

import (
	"strings"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/lsp"
	"terragrunt-ls/internal/tg/completion"
	"terragrunt-ls/internal/tg/definition"
	"terragrunt-ls/internal/tg/hover"
	"terragrunt-ls/internal/tg/store"

	"github.com/gruntwork-io/terragrunt/config"
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

func (s *State) OpenDocument(l logger.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	l.Debug(
		"Opening document",
		"uri", docURI,
		"text", text,
	)

	return s.updateState(l, docURI, text)
}

func (s *State) UpdateDocument(l logger.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	l.Debug(
		"Updating document",
		"uri", docURI,
		"text", text,
	)

	return s.updateState(l, docURI, text)
}

func (s *State) updateState(l logger.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	cfg, diags := parseTerragruntBuffer(docURI.Filename(), text)

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

	s.Configs[docURI.Filename()] = store.Store{
		Cfg:      cfg,
		CfgAsCty: cfgAsCty,
		Document: text,
	}

	return diags
}

func (s *State) Hover(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.HoverResponse {
	store := s.Configs[docURI.Filename()]

	l.Debug(
		"Hovering over character",
		"uri", docURI,
		"position", position,
	)

	l.Debug(
		"Config",
		"uri", docURI,
		"config", store.Cfg,
	)

	word, context := hover.GetHoverTargetWithContext(l, store, position)

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
		if store.Cfg == nil {
			return newEmptyHoverResponse(id)
		}

		if _, ok := store.Cfg.Locals[word]; !ok {
			return newEmptyHoverResponse(id)
		}

		if store.CfgAsCty.IsNull() {
			return newEmptyHoverResponse(id)
		}

		locals := store.CfgAsCty.GetAttr("locals")
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
					Value: wrapAsHCLCodeFence(strings.TrimSpace(string(f.Bytes()))),
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

func wrapAsHCLCodeFence(s string) string {
	return "```hcl\n" + s + "\n```"
}

func (s *State) Definition(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	store := s.Configs[docURI.Filename()]

	l.Debug(
		"Definition requested",
		"uri", docURI,
		"position", position,
	)

	target, context := definition.GetDefinitionTargetWithContext(l, store, position)

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
			"store", store,
		)

		if store.Cfg == nil {
			return newEmptyDefinitionResponse(id, docURI, position)
		}

		l.Debug(
			"Includes",
			"includes", store.Cfg.ProcessedIncludes,
		)

		for _, include := range store.Cfg.ProcessedIncludes {
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
	store := s.Configs[docURI.Filename()]

	l.Debug(
		"Formatting requested",
		"uri", docURI,
	)

	formatted := hclwrite.Format([]byte(store.Document))

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
					End: getEndOfDocument(store.Document),
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
