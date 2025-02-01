package tg

import (
	"log"

	"github.com/gruntwork-io/terragrunt/config"
	"go.lsp.dev/protocol"
)

type State struct {
	// Map of file names to contents
	Documents map[string]*config.TerragruntConfig
}

func NewState() State {
	return State{Documents: map[string]*config.TerragruntConfig{}}
}

func (s *State) OpenDocument(l *log.Logger, uri protocol.DocumentURI, text string) []protocol.Diagnostic {
	cfg, diags := parseTerragruntBuffer(uri.Filename(), text)

	l.Printf("Opened document: %s", uri.Filename())
	l.Printf("Config: %v", cfg)

	s.Documents[uri.Filename()] = cfg

	return diags
}

func (s *State) UpdateDocument(l *log.Logger, uri protocol.DocumentURI, text string) []protocol.Diagnostic {
	cfg, diags := parseTerragruntBuffer(uri.Filename(), text)

	l.Printf("Updated document: %s", uri.Filename())
	l.Printf("Config: %v", cfg)

	s.Documents[uri.Filename()] = cfg

	return diags
}

// func (s *State) Hover(id int, uri string, position lsp.Position) lsp.HoverResponse {
// 	// In real life, this would look up the type in our type analysis code...
//
// 	document := s.Documents[uri]
//
// 	return lsp.HoverResponse{
// 		Response: lsp.Response{
// 			RPC: "2.0",
// 			ID:  &id,
// 		},
// 		Result: lsp.HoverResult{
// 			Contents: fmt.Sprintf("File: %s, Characters: %d", uri, len(document)),
// 		},
// 	}
// }
//
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
