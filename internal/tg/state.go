package tg

import (
	"os"
	"path/filepath"
	"strings"
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/lsp"
	"terragrunt-ls/internal/tg/completion"
	"terragrunt-ls/internal/tg/definition"
	"terragrunt-ls/internal/tg/hover"
	"terragrunt-ls/internal/tg/rename"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"

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
	// Ignore errors from AST indexing since we'll get the same errors from the Terragrunt parser just below
	ast, _ := ast.ParseHCLFile(docURI.Filename(), []byte(text))

	cfg, diags := ParseTerragruntBuffer(l, docURI.Filename(), text)

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
		AST:      ast,
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
	case definition.DefinitionContextDependency:
		l.Debug(
			"Store content",
			"store", store,
		)

		if store.Cfg == nil {
			return newEmptyDefinitionResponse(id, docURI, position)
		}

		l.Debug(
			"Dependencies",
			"dependencies", store.Cfg.TerragruntDependencies,
		)

		for _, dep := range store.Cfg.TerragruntDependencies {
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

func (s *State) PrepareRename(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.PrepareRenameResponse {
	store, ok := s.Configs[docURI.Filename()]
	if !ok {
		l.Debug("No config found for document", "uri", docURI)
		return lsp.PrepareRenameResponse{
			Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
			Result:   nil,
		}
	}

	l.Debug("Prepare rename requested", "uri", docURI, "position", position)

	// Check if the identifier at this position is actually renameable
	target, context := rename.GetRenameTargetWithContext(l, store, position)
	if target == "" || context == rename.RenameContextNull {
		l.Debug("No renameable identifier at position")
		return lsp.PrepareRenameResponse{
			Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
			Result:   nil,
		}
	}

	// Get the word range at the cursor position
	wordRange := text.GetCursorWordRange(store.Document, position)
	word := text.GetCursorWord(store.Document, position)

	if wordRange == nil || word == "" {
		l.Debug("No word found at position for rename")
		return lsp.PrepareRenameResponse{
			Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
			Result:   nil,
		}
	}

	l.Debug("Prepare rename result", "word", word, "range", wordRange)

	return lsp.PrepareRenameResponse{
		Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
		Result: &lsp.PrepareRenameResult{
			Range:       *wordRange,
			Placeholder: word,
		},
	}
}

func (s *State) TextDocumentRename(l logger.Logger, id int, docURI protocol.DocumentURI, position protocol.Position, newName string) lsp.RenameResponse {
	store, ok := s.Configs[docURI.Filename()]
	if !ok {
		l.Error("No config found for document", "uri", docURI)
		return newEmptyRenameResponse(id)
	}

	l.Debug("Rename requested", "uri", docURI, "position", position, "newName", newName)

	// Determine what we're renaming
	target, context := rename.GetRenameTargetWithContext(l, store, position)
	l.Debug("Rename target discovered", "target", target, "context", context)

	// If nothing valid to rename, return null
	if target == "" || context == rename.RenameContextNull {
		l.Debug("No renameable identifier at position")
		return newEmptyRenameResponse(id)
	}

	// Normalize the new name (strip "local." prefix for local variables)
	normalizedNewName := normalizeNewName(l, newName, context)

	// Find all occurrences of the identifier
	occurrences := rename.FindAllOccurrences(l, store.Document, target, context)
	if len(occurrences) == 0 {
		l.Debug("No occurrences found to rename")
		return newEmptyRenameResponse(id)
	}

	// Create text edits for all occurrences
	edits := createTextEdits(occurrences, normalizedNewName)
	l.Debug("Rename edits created", "count", len(edits))

	return lsp.RenameResponse{
		Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
		Result: &protocol.WorkspaceEdit{
			Changes: map[protocol.DocumentURI][]protocol.TextEdit{docURI: edits},
		},
	}
}

func newEmptyRenameResponse(id int) lsp.RenameResponse {
	return lsp.RenameResponse{
		Response: lsp.Response{RPC: lsp.RPCVersion, ID: &id},
		Result:   nil,
	}
}

func normalizeNewName(l logger.Logger, newName, context string) string {
	if context == rename.RenameContextLocal && strings.HasPrefix(newName, "local.") {
		normalized := strings.TrimPrefix(newName, "local.")
		l.Debug("Stripped local. prefix from new name", "normalized", normalized)
		return normalized
	}
	return newName
}

func createTextEdits(ranges []protocol.Range, newText string) []protocol.TextEdit {
	edits := make([]protocol.TextEdit, len(ranges))
	for i, r := range ranges {
		edits[i] = protocol.TextEdit{Range: r, NewText: newText}
	}
	return edits
}

func getEndOfDocument(doc string) protocol.Position {
	lines := strings.Split(doc, "\n")

	return protocol.Position{
		Line:      uint32(len(lines) - 1),
		Character: uint32(len(lines[len(lines)-1])),
	}
}
