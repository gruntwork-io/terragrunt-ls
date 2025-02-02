package tg

import (
	"log"
	"strings"
	"terragrunt-ls/lsp"
	"terragrunt-ls/tg/hover"
	"terragrunt-ls/tg/store"

	"github.com/gruntwork-io/terragrunt/config"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"go.lsp.dev/protocol"
)

type State struct {
	// Map of file names to Terragrunt configs
	Configs map[string]store.Store
}

func NewState() State {
	return State{Configs: map[string]store.Store{}}
}

func (s *State) OpenDocument(l *log.Logger, uri protocol.DocumentURI, text string) []protocol.Diagnostic {
	cfg, diags := parseTerragruntBuffer(uri.Filename(), text)

	l.Printf("Opened document: %s", uri.Filename())
	l.Printf("Config: %v", cfg)

	cfgAsCty := cty.NilVal

	if cfg != nil {
		if converted, err := config.TerragruntConfigAsCty(cfg); err == nil {
			cfgAsCty = converted
		}
	}

	s.Configs[uri.Filename()] = store.Store{
		Cfg:      cfg,
		CfgAsCty: cfgAsCty,
		Document: text,
	}

	return diags
}

func (s *State) UpdateDocument(l *log.Logger, uri protocol.DocumentURI, text string) []protocol.Diagnostic {
	cfg, diags := parseTerragruntBuffer(uri.Filename(), text)

	l.Printf("Updated document: %s", uri.Filename())
	l.Printf("Config: %v", cfg)

	cfgAsCty := cty.NilVal

	if cfg != nil {
		if converted, err := config.TerragruntConfigAsCty(cfg); err == nil {
			cfgAsCty = converted
		}
	}

	s.Configs[uri.Filename()] = store.Store{
		Cfg:      cfg,
		CfgAsCty: cfgAsCty,
		Document: text,
	}

	return diags
}

func (s *State) Hover(l *log.Logger, id int, uri protocol.DocumentURI, position protocol.Position) lsp.HoverResponse {
	store := s.Configs[uri.Filename()]

	l.Printf("Hovering over %s at %d:%d", uri, position.Line, position.Character)
	l.Printf("Config: %v", store.Document)

	word, context := hover.GetHoverTargetWithContext(l, store, position)

	l.Printf("Word: %s, Context: %s", word, context)

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

// func (s *State) Definition(id int, uri string, position lsp.Position) lsp.DefinitionResponse {
// 	// In real life, this would look up the definition
//
// 	return lsp.DefinitionResponse{
// 		Response: lsp.Response{
// 			RPC: "2.0",
// 			ID:  &id,
// 		},
// 		Result: lsp.Location{
// 			URI: uri,
// 			Range: lsp.Range{
// 				Start: lsp.Position{
// 					Line:      position.Line - 1,
// 					Character: 0,
// 				},
// 				End: lsp.Position{
// 					Line:      position.Line - 1,
// 					Character: 0,
// 				},
// 			},
// 		},
// 	}
// }
// func (s *State) TextDocumentCodeAction(id int, uri string) lsp.TextDocumentCodeActionResponse {
// 	text := s.Documents[uri]
//
// 	actions := []lsp.CodeAction{}
// 	for row, line := range strings.Split(text, "\n") {
// 		idx := strings.Index(line, "VS Code")
// 		if idx >= 0 {
// 			replaceChange := map[string][]lsp.TextEdit{}
// 			replaceChange[uri] = []lsp.TextEdit{
// 				{
// 					Range:   LineRange(row, idx, idx+len("VS Code")),
// 					NewText: "Neovim",
// 				},
// 			}
//
// 			actions = append(actions, lsp.CodeAction{
// 				Title: "Replace VS C*de with a superior editor",
// 				Edit:  &lsp.WorkspaceEdit{Changes: replaceChange},
// 			})
//
// 			censorChange := map[string][]lsp.TextEdit{}
// 			censorChange[uri] = []lsp.TextEdit{
// 				{
// 					Range:   LineRange(row, idx, idx+len("VS Code")),
// 					NewText: "VS C*de",
// 				},
// 			}
//
// 			actions = append(actions, lsp.CodeAction{
// 				Title: "Censor to VS C*de",
// 				Edit:  &lsp.WorkspaceEdit{Changes: censorChange},
// 			})
// 		}
// 	}
//
// 	response := lsp.TextDocumentCodeActionResponse{
// 		Response: lsp.Response{
// 			RPC: "2.0",
// 			ID:  &id,
// 		},
// 		Result: actions,
// 	}
//
// 	return response
// }
//
// func (s *State) TextDocumentCompletion(id int, uri string) lsp.CompletionResponse {
//
// 	// Ask your static analysis tools to figure out good completions
// 	items := []lsp.CompletionItem{
// 		{
// 			Label:         "Neovim (BTW)",
// 			Detail:        "Very cool editor",
// 			Documentation: "Fun to watch in videos. Don't forget to like & subscribe to streamers using it :)",
// 		},
// 	}
//
// 	response := lsp.CompletionResponse{
// 		Response: lsp.Response{
// 			RPC: "2.0",
// 			ID:  &id,
// 		},
// 		Result: items,
// 	}
//
// 	return response
// }
//
// func LineRange(line, start, end int) lsp.Range {
// 	return lsp.Range{
// 		Start: lsp.Position{
// 			Line:      line,
// 			Character: start,
// 		},
// 		End: lsp.Position{
// 			Line:      line,
// 			Character: end,
// 		},
// 	}
// }
