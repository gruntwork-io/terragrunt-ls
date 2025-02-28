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

func (s *State) OpenDocument(l *logger.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	l.Debugf("Opening document: %s", docURI.Filename())

	return s.updateState(l, docURI, text)
}

func (s *State) UpdateDocument(l *logger.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	l.Debugf("Updating document: %s", docURI.Filename())

	return s.updateState(l, docURI, text)
}

func (s *State) updateState(l *logger.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	cfg, diags := parseTerragruntBuffer(docURI.Filename(), text)

	l.Debugf("Config: %v", cfg)

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

func (s *State) Hover(l *logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.HoverResponse {
	store := s.Configs[docURI.Filename()]

	l.Debugf("Hovering over %s at %d:%d", docURI, position.Line, position.Character)
	l.Debugf("Config: %v", store.Document)

	word, context := hover.GetHoverTargetWithContext(l, store, position)

	l.Debugf("Hovering over %s with context %s", word, context)

	if word == "" {
		return buildEmptyHoverResponse(id)
	}

	switch context {
	case hover.HoverContextLocal:
		if store.Cfg == nil {
			return buildEmptyHoverResponse(id)
		}

		if _, ok := store.Cfg.Locals[word]; !ok {
			return buildEmptyHoverResponse(id)
		}

		if store.CfgAsCty.IsNull() {
			return buildEmptyHoverResponse(id)
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

	return buildEmptyHoverResponse(id)
}

func buildEmptyHoverResponse(id int) lsp.HoverResponse {
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

func (s *State) Definition(l *logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	store := s.Configs[docURI.Filename()]

	l.Debugf("Definition for %s at %d:%d", docURI, position.Line, position.Character)

	target, context := definition.GetDefinitionTargetWithContext(l, store, position)

	l.Debugf("Definition for %s with context %s", target, context)

	if target == "" {
		return buildEmptyDefinitionResponse(id, docURI, position)
	}

	switch context {
	case definition.DefinitionContextInclude:
		l.Debugf("Store: %v", store)

		if store.Cfg == nil {
			return buildEmptyDefinitionResponse(id, docURI, position)
		}

		l.Debugf("Includes: %v", store.Cfg.ProcessedIncludes)

		for _, include := range store.Cfg.ProcessedIncludes {
			if include.Name == target {
				l.Debugf("Jumping to %s %s", include.Name, include.Path)

				defURI := uri.File(include.Path)

				l.Debugf("URI: %s", defURI)

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

	return buildEmptyDefinitionResponse(id, docURI, position)
}

func buildEmptyDefinitionResponse(id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
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

func (s *State) TextDocumentCompletion(l *logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.CompletionResponse {
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
