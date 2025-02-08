// package tg
//
// import (
// 	"log"
// 	"strings"
// 	"terragrunt-ls/lsp"
// 	"terragrunt-ls/tg/completion"
// 	"terragrunt-ls/tg/definition"
// 	"terragrunt-ls/tg/hover"
// 	"terragrunt-ls/tg/store"
//
// 	"github.com/gruntwork-io/terragrunt/config"
// 	"github.com/hashicorp/hcl/v2/hclwrite"
// 	"github.com/zclconf/go-cty/cty"
// 	"go.lsp.dev/protocol"
// 	"go.lsp.dev/uri"
// )
//
// type State struct {
// 	// Map of file names to Terragrunt configs
// 	Configs map[string]store.Store
// }
//
// func NewState() State {
// 	return State{Configs: map[string]store.Store{}}
// }
//
// func (s *State) OpenDocument(l *log.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
// 	l.Printf("Opening document: %s", docURI.Filename())
//
// 	return s.updateState(l, docURI, text)
// }
//
// func (s *State) UpdateDocument(l *log.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
// 	l.Printf("Updating document: %s", docURI.Filename())
//
// 	return s.updateState(l, docURI, text)
// }
//
// func (s *State) updateState(l *log.Logger, docURI protocol.DocumentURI, text string) []protocol.Diagnostic {
// 	cfg, diags := parseTerragruntBuffer(docURI.Filename(), text)
//
// 	l.Printf("Config: %v", cfg)
//
// 	cfgAsCty := cty.NilVal
//
// 	if cfg != nil {
// 		if converted, err := config.TerragruntConfigAsCty(cfg); err == nil {
// 			cfgAsCty = converted
// 		}
// 	}
//
// 	s.Configs[docURI.Filename()] = store.Store{
// 		Cfg:      cfg,
// 		CfgAsCty: cfgAsCty,
// 		Document: text,
// 	}
//
// 	return diags
// }
//
// func (s *State) Hover(l *log.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.HoverResponse {
// 	store := s.Configs[docURI.Filename()]
//
// 	l.Printf("Hovering over %s at %d:%d", docURI, position.Line, position.Character)
// 	l.Printf("Config: %v", store.Document)
//
// 	word, context := hover.GetHoverTargetWithContext(l, store, position)
//
// 	l.Printf("Word: %s, Context: %s", word, context)
//
// 	switch context {
// 	case hover.HoverContextLocal:
// 		if store.Cfg == nil {
// 			return buildEmptyHoverResponse(id)
// 		}
//
// 		if _, ok := store.Cfg.Locals[word]; !ok {
// 			return buildEmptyHoverResponse(id)
// 		}
//
// 		if store.CfgAsCty.IsNull() {
// 			return buildEmptyHoverResponse(id)
// 		}
//
// 		locals := store.CfgAsCty.GetAttr("locals")
// 		localVal := locals.GetAttr(word)
//
// 		f := hclwrite.NewEmptyFile()
// 		rootBody := f.Body()
// 		rootBody.SetAttributeValue(word, localVal)
//
// 		return lsp.HoverResponse{
// 			Response: lsp.Response{
// 				RPC: lsp.RPCVersion,
// 				ID:  &id,
// 			},
// 			Result: lsp.HoverResult{
// 				Contents: protocol.MarkupContent{
// 					Kind:  protocol.Markdown,
// 					Value: wrapAsHCLCodeFence(strings.TrimSpace(string(f.Bytes()))),
// 				},
// 			},
// 		}
//
// 	case hover.HoverContextNull:
// 		return buildEmptyHoverResponse(id)
// 	}
//
// 	return buildEmptyHoverResponse(id)
// }
//
// func buildEmptyHoverResponse(id int) lsp.HoverResponse {
// 	return lsp.HoverResponse{
// 		Response: lsp.Response{
// 			RPC: lsp.RPCVersion,
// 			ID:  &id,
// 		},
// 	}
// }
//
// func wrapAsHCLCodeFence(s string) string {
// 	return "```hcl\n" + s + "\n```"
// }
//
// func (s *State) Definition(l *log.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
// 	store := s.Configs[docURI.Filename()]
//
// 	l.Printf("Jumping to definition from %s at %d:%d", docURI, position.Line, position.Character)
//
// 	target, context := definition.GetDefinitionTargetWithContext(l, store, position)
//
// 	l.Printf("Target: %s, Context: %s", target, context)
//
// 	switch context {
// 	case definition.DefinitionContextInclude:
// 		if store.Cfg == nil {
// 			return buildEmptyDefinitionResponse(id, docURI, position)
// 		}
//
// 		for _, include := range store.Cfg.ProcessedIncludes {
// 			if include.Name == target {
// 				l.Printf("Jumping to %s %s", include.Name, include.Path)
//
// 				defURI := uri.File(include.Path)
//
// 				l.Printf("URI: %s", defURI)
//
// 				return lsp.DefinitionResponse{
// 					Response: lsp.Response{
// 						RPC: lsp.RPCVersion,
// 						ID:  &id,
// 					},
// 					Result: protocol.Location{
// 						URI: defURI,
// 						Range: protocol.Range{
// 							Start: protocol.Position{
// 								Line:      0,
// 								Character: 0,
// 							},
// 							End: protocol.Position{
// 								Line:      0,
// 								Character: 0,
// 							},
// 						},
// 					},
// 				}
// 			}
// 		}
//
// 	case definition.DefinitionContextNull:
// 		return buildEmptyDefinitionResponse(id, docURI, position)
// 	}
//
// 	return buildEmptyDefinitionResponse(id, docURI, position)
// }
//
// // NOTE: I think I'm supposed to be able to return a null response here,
// // but I'm getting errors when I try to do that.
// // Instead, I'm returning the same location I started from.
// func buildEmptyDefinitionResponse(id int, docURI protocol.DocumentURI, position protocol.Position) lsp.DefinitionResponse {
// 	return lsp.DefinitionResponse{
// 		Response: lsp.Response{
// 			RPC: lsp.RPCVersion,
// 			ID:  &id,
// 		},
// 		Result: protocol.Location{
// 			URI: docURI,
// 			Range: protocol.Range{
// 				Start: protocol.Position{
// 					Line:      position.Line,
// 					Character: position.Character,
// 				},
// 				End: protocol.Position{
// 					Line:      position.Line,
// 					Character: position.Character,
// 				},
// 			},
// 		},
// 	}
// }
//
// func (s *State) TextDocumentCompletion(l *log.Logger, id int, docURI protocol.DocumentURI, position protocol.Position) lsp.CompletionResponse {
// 	items := completion.GetCompletions(l, s.Configs[docURI.Filename()], position)
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

// Let's add some tests for this.

package tg_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gruntwork-io/terragrunt/codegen"
	"github.com/gruntwork-io/terragrunt/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	// "go.lsp.dev/uri"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"terragrunt-ls/internal/lsp"
	"terragrunt-ls/internal/tg"
)

func TestNewState(t *testing.T) {
	t.Parallel()

	state := tg.NewState()

	assert.NotNil(t, state.Configs)
}

func TestState_OpenDocument(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	_, err := createFile(tmpDir, "root.hcl", "")
	require.NoError(t, err)

	rootPath := filepath.Join(tmpDir, "root.hcl")

	// rootURI := uri.File(rootPath)

	unitDir := filepath.Join(tmpDir, "foo")

	err = os.MkdirAll(unitDir, 0755)
	require.NoError(t, err)

	// Create the URI for the unit file
	unitPath := filepath.Join(unitDir, "bar.hcl")

	unitURI := uri.File(unitPath)

	tc := []struct {
		name     string
		document string
		expected *config.TerragruntConfig
	}{
		{
			name:     "empty document",
			document: "",
			expected: &config.TerragruntConfig{
				GenerateConfigs:   map[string]codegen.GenerateConfig{},
				ProcessedIncludes: config.IncludeConfigsMap{},
			},
		},
		{
			name: "simple locals",
			document: `locals {
	foo = "bar"
}`,
			expected: &config.TerragruntConfig{
				Locals: map[string]interface{}{
					"foo": "bar",
				},
				GenerateConfigs:   map[string]codegen.GenerateConfig{},
				ProcessedIncludes: config.IncludeConfigsMap{},
				FieldsMetadata: map[string]map[string]interface{}{
					"locals-foo": {
						"found_in_file": unitPath,
					},
				},
			},
		},
		{
			name: "multiple locals",
			document: `locals {
	foo = "bar"
	baz = "qux"
}`,
			expected: &config.TerragruntConfig{
				Locals: map[string]interface{}{
					"baz": "qux",
					"foo": "bar",
				},
				GenerateConfigs:   map[string]codegen.GenerateConfig{},
				ProcessedIncludes: config.IncludeConfigsMap{},
				FieldsMetadata: map[string]map[string]interface{}{
					"locals-baz": {
						"found_in_file": unitPath,
					},
					"locals-foo": {
						"found_in_file": unitPath,
					},
				},
			},
		},
		{
			name: "root include",
			document: `include "root" {
	path = find_in_parent_folders("root.hcl")
}`,
			expected: &config.TerragruntConfig{
				GenerateConfigs: map[string]codegen.GenerateConfig{},
				ProcessedIncludes: config.IncludeConfigsMap{
					"root": {
						Name: "root",
						Path: rootPath,
					},
				},
				TerragruntDependencies: config.Dependencies{},
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := tg.NewState()

			l := newTestLogger(t)

			diags := state.OpenDocument(l, unitURI, tt.document)
			require.Empty(t, diags)

			assert.Len(t, state.Configs, 1)

			assert.Equal(t, tt.expected, state.Configs[unitPath].Cfg)
		})
	}
}

func TestState_UpdateDocument(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name            string
		document        string
		expected        map[string]interface{}
		updated         string
		expectedUpdated map[string]interface{}
	}{
		{
			name:     "empty document",
			document: "",
		},
		{
			name: "simple locals",
			document: `locals {
	foo = "bar"
}`,
			expected: map[string]interface{}{
				"foo": "bar",
			},
			updated: `locals {
	foo = "baz"
}`,
			expectedUpdated: map[string]interface{}{
				"foo": "baz",
			},
		},
		{
			name: "multiple locals",
			document: `locals {
	foo = "bar"
	baz = "qux"
}`,
			expected: map[string]interface{}{
				"foo": "bar",
				"baz": "qux",
			},
			updated: `locals {
	foo = "baz"
	baz = "qux"
}`,
			expectedUpdated: map[string]interface{}{
				"foo": "baz",
				"baz": "qux",
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := tg.NewState()

			l := newTestLogger(t)

			diags := state.OpenDocument(l, "file:///foo/bar.hcl", tt.document)
			assert.Empty(t, diags)

			require.Len(t, state.Configs, 1)

			if len(tt.expected) != 0 {
				assert.Equal(t, tt.expected, state.Configs["/foo/bar.hcl"].Cfg.Locals)
			}

			diags = state.UpdateDocument(l, "file:///foo/bar.hcl", tt.updated)
			assert.Empty(t, diags)

			assert.Len(t, state.Configs, 1)

			if len(tt.expectedUpdated) != 0 {
				assert.Equal(t, tt.expectedUpdated, state.Configs["/foo/bar.hcl"].Cfg.Locals)
			}
		})
	}
}

func TestState_Hover(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		document string
		position protocol.Position
		expected lsp.HoverResponse
	}{
		{
			name: "simple locals",
			document: `locals {
	foo = "bar"
	bar = local.foo
}`,
			position: protocol.Position{
				Line:      2,
				Character: 15,
			},
			expected: lsp.HoverResponse{
				Response: lsp.Response{
					RPC: "2.0",
					ID:  pointerOfInt(1),
				},
				Result: lsp.HoverResult{
					Contents: protocol.MarkupContent{
						Kind:  protocol.Markdown,
						Value: "```hcl\nfoo = \"bar\"\n```",
					},
				},
			},
		},
		{
			name: "interpolated locals",
			document: `locals {
	foo = "bar"
	baz = "${local.foo}-baz"
	qux = local.baz
}`,
			position: protocol.Position{
				Line:      3,
				Character: 15,
			},
			expected: lsp.HoverResponse{
				Response: lsp.Response{
					RPC: "2.0",
					ID:  pointerOfInt(1),
				},
				Result: lsp.HoverResult{
					Contents: protocol.MarkupContent{
						Kind:  protocol.Markdown,
						Value: "```hcl\nbaz = \"bar-baz\"\n```",
					},
				},
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := tg.NewState()

			l := newTestLogger(t)

			diags := state.OpenDocument(l, "file:///foo/bar.hcl", tt.document)
			assert.Empty(t, diags)

			require.Len(t, state.Configs, 1)

			hover := state.Hover(l, 1, "file:///foo/bar.hcl", tt.position)
			assert.Equal(t, tt.expected, hover)
		})
	}
}

func TestState_Definition(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	_, err := createFile(tmpDir, "root.hcl", "")
	require.NoError(t, err)

	rootURI := uri.File(filepath.Join(tmpDir, "root.hcl"))

	unitDir := filepath.Join(tmpDir, "foo")

	err = os.MkdirAll(unitDir, 0755)
	require.NoError(t, err)

	// Create the URI for the unit file
	unitPath := filepath.Join(unitDir, "bar.hcl")

	unitURI := uri.File(unitPath)

	tc := []struct {
		name     string
		document string
		position protocol.Position
		expected lsp.DefinitionResponse
	}{
		{
			name: "nothing to jump to",
			document: `locals {
	foo = "bar"
	bar = local.foo
}`,
			position: protocol.Position{
				Line:      0,
				Character: 0,
			},
			expected: lsp.DefinitionResponse{
				Response: lsp.Response{
					RPC: "2.0",
					ID:  pointerOfInt(1),
				},
				Result: protocol.Location{
					URI: unitURI,
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
			},
		},
		{
			name: "go to root include",
			document: `include "root" {
	path = find_in_parent_folders("root.hcl")
}`,
			position: protocol.Position{
				Line:      0,
				Character: 0,
			},
			expected: lsp.DefinitionResponse{
				Response: lsp.Response{
					RPC: "2.0",
					ID:  pointerOfInt(1),
				},
				Result: protocol.Location{
					URI: rootURI,
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
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := tg.NewState()

			l := newTestLogger(t)

			diags := state.OpenDocument(l, unitURI, tt.document)
			assert.Empty(t, diags)

			require.Len(t, state.Configs, 1)

			definition := state.Definition(l, 1, unitURI, tt.position)
			assert.Equal(t, tt.expected, definition)
		})
	}
}

func newTestLogger(t *testing.T) *zap.SugaredLogger {
	t.Helper()

	l := zaptest.NewLogger(t)
	return zap.New(l.Core(), zap.AddCaller()).Sugar()
}

func pointerOfInt(i int) *int {
	return &i
}

func createFile(dir, name, content string) (string, error) {
	return createFileWithMode(dir, name, content, 0644)
}

func createFileWithMode(dir, name, content string, mode os.FileMode) (string, error) {
	path := filepath.Join(dir, name)

	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		return "", err
	}

	return path, nil
}
