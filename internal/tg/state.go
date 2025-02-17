package tg

import (
	"strings"
	"terragrunt-ls/internal/ast"
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
	"go.uber.org/zap"
)

type State struct {
	// Map of file names to Terragrunt configs
	Configs map[string]store.Store
}

func NewState() State {
	return State{Configs: map[string]store.Store{}}
}

func (s *State) OpenDocument(l *zap.SugaredLogger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	l.Debugf("Opening document: %s", docURI.Filename())

	return s.updateState(l, docURI, text)
}

func (s *State) UpdateDocument(l *zap.SugaredLogger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	l.Debugf("Updating document: %s", docURI.Filename())

	return s.updateState(l, docURI, text)
}

func (s *State) updateState(l *zap.SugaredLogger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
	ast, err := ast.IndexFileAST(docURI.Filename(), []byte(text))
	if err != nil {
		l.Errorf("Error indexing AST: %v", err)
	}

	cfg, diags := parseTerragruntBuffer(docURI.Filename(), text)

	l.Debugf("Config: %v", cfg)

	cfgAsCty := cty.NilVal

	if cfg != nil {
		if converted, err := config.TerragruntConfigAsCty(cfg); err == nil {
			cfgAsCty = converted
		}
	}

	s.Configs[docURI.Filename()] = store.Store{
		AST:      ast,
		Cfg:      cfg,
		CfgAsCty: cfgAsCty,
		Document: text,
	}

	return diags
}

func (s *State) Hover(l *zap.SugaredLogger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.HoverResponse {
	store := s.Configs[docURI.Filename()]

	l.Debugf("Hovering over %s at %d:%d", docURI, position.Line, position.Character)

	word, context := hover.GetHoverTargetWithContext(l, store, position)

	l.Debugf("Word: %s, Context: %s", word, context)

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

	case hover.HoverContextNull:
		return buildEmptyHoverResponse(id)
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

func (s *State) Definition(l *zap.SugaredLogger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	store := s.Configs[docURI.Filename()]

	l.Debugf("Jumping to definition from %s at %d:%d", docURI, position.Line, position.Character)

	target, context := definition.GetDefinitionTargetWithContext(l, store, position)

	l.Debugf("Target: %s, Context: %s", target, context)

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

	case definition.DefinitionContextNull:
		return buildEmptyDefinitionResponse(id, docURI, position)
	}

	return buildEmptyDefinitionResponse(id, docURI, position)
}

// NOTE: I think I'm supposed to be able to return a null response here,
// but I'm getting errors when I try to do that.
// Instead, I'm returning the same location I started from.
func buildEmptyDefinitionResponse(id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
	return lsp.DefinitionResponse{
		Response: lsp.Response{
			RPC: lsp.RPCVersion,
			ID:  &id,
		},
		Result: protocol.Location{
			URI: docURI,
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      position.Line,
					Character: position.Character,
				},
				End: protocol.Position{
					Line:      position.Line,
					Character: position.Character,
				},
			},
		},
	}
}

func (s *State) TextDocumentCompletion(l *zap.SugaredLogger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.CompletionResponse {
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
