package tg_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/terragrunt/codegen"
	"github.com/gruntwork-io/terragrunt/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"terragrunt-ls/internal/lsp"
	"terragrunt-ls/internal/testutils"
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

	_, err := testutils.CreateFile(tmpDir, "root.hcl", "")
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
				Locals: map[string]any{
					"foo": "bar",
				},
				GenerateConfigs:   map[string]codegen.GenerateConfig{},
				ProcessedIncludes: config.IncludeConfigsMap{},
				FieldsMetadata: map[string]map[string]any{
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
				Locals: map[string]any{
					"baz": "qux",
					"foo": "bar",
				},
				GenerateConfigs:   map[string]codegen.GenerateConfig{},
				ProcessedIncludes: config.IncludeConfigsMap{},
				FieldsMetadata: map[string]map[string]any{
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

			l := testutils.NewTestLogger(t)

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
		expected        map[string]any
		updated         string
		expectedUpdated map[string]any
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
			expected: map[string]any{
				"foo": "bar",
			},
			updated: `locals {
	foo = "baz"
}`,
			expectedUpdated: map[string]any{
				"foo": "baz",
			},
		},
		{
			name: "multiple locals",
			document: `locals {
	foo = "bar"
	baz = "qux"
}`,
			expected: map[string]any{
				"foo": "bar",
				"baz": "qux",
			},
			updated: `locals {
	foo = "baz"
	baz = "qux"
}`,
			expectedUpdated: map[string]any{
				"foo": "baz",
				"baz": "qux",
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := tg.NewState()

			l := testutils.NewTestLogger(t)

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
					ID:  testutils.PointerOfInt(1),
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
					ID:  testutils.PointerOfInt(1),
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

			l := testutils.NewTestLogger(t)

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

	_, err := testutils.CreateFile(tmpDir, "root.hcl", "")
	require.NoError(t, err)

	rootURI := uri.File(filepath.Join(tmpDir, "root.hcl"))

	// Create a vpc directory
	vpcDir := filepath.Join(tmpDir, "vpc")
	err = os.MkdirAll(vpcDir, 0755)
	require.NoError(t, err)

	// Create a terragrunt.hcl file in the vpc directory
	_, err = testutils.CreateFile(vpcDir, "terragrunt.hcl", "")
	require.NoError(t, err)

	vpcURI := uri.File(filepath.Join(vpcDir, "terragrunt.hcl"))

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
					ID:  testutils.PointerOfInt(1),
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
				Line:      1,
				Character: 8,
			},
			expected: lsp.DefinitionResponse{
				Response: lsp.Response{
					RPC: "2.0",
					ID:  testutils.PointerOfInt(1),
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
		{
			name: "go to dependency",
			document: `dependency "vpc" {
    config_path = "../vpc"
}`,
			position: protocol.Position{
				Line:      1,
				Character: 18,
			},
			expected: lsp.DefinitionResponse{
				Response: lsp.Response{
					RPC: "2.0",
					ID:  testutils.PointerOfInt(1),
				},
				Result: protocol.Location{
					URI: vpcURI,
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

			l := testutils.NewTestLogger(t)

			diags := state.OpenDocument(l, unitURI, tt.document)
			assert.Empty(t, diags)

			require.Len(t, state.Configs, 1)

			definition := state.Definition(l, 1, unitURI, tt.position)
			assert.Equal(t, tt.expected, definition)
		})
	}
}

func TestState_TextDocumentCompletion(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name              string
		initial           string
		document          string
		position          protocol.Position
		expected          lsp.CompletionResponse
		expectDiagnostics bool
	}{
		{
			name:     "complete dep",
			document: "dep",
			position: protocol.Position{
				Line:      0,
				Character: 3,
			},
			expectDiagnostics: true,
			expected: lsp.CompletionResponse{
				Response: lsp.Response{
					RPC: "2.0",
					ID:  testutils.PointerOfInt(1),
				},
				Result: []protocol.CompletionItem{
					{
						Label: "dependency",
						Documentation: protocol.MarkupContent{
							Kind:  protocol.Markdown,
							Value: "# dependency\nThe dependency block is used to configure unit dependencies.\nEach dependency block exposes outputs of the dependency unit as variables you can reference in dependent unit configuration.",
						},
						Kind:             protocol.CompletionItemKindClass,
						InsertTextFormat: protocol.InsertTextFormatSnippet,
						TextEdit: &protocol.TextEdit{
							Range: protocol.Range{
								Start: protocol.Position{Line: 0, Character: 0},
								End:   protocol.Position{Line: 0, Character: 3},
							},
							NewText: `dependency "${1}" {
	config_path = "${2}"
}`,
						},
					},
					{
						Label: "dependencies",
						Documentation: protocol.MarkupContent{
							Kind:  protocol.Markdown,
							Value: "# dependencies\nThe dependencies block is used to enumerate all the Terragrunt units that need to be applied before this unit.",
						},
						Kind:             protocol.CompletionItemKindClass,
						InsertTextFormat: protocol.InsertTextFormatSnippet,
						TextEdit: &protocol.TextEdit{
							Range: protocol.Range{
								Start: protocol.Position{Line: 0, Character: 0},
								End:   protocol.Position{Line: 0, Character: 3},
							},
							NewText: `dependencies {
	paths = ["${1}"]
}`,
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
			l := testutils.NewTestLogger(t)

			diags := state.OpenDocument(l, "file:///test.hcl", tt.document)
			if tt.expectDiagnostics {
				require.NotEmpty(t, diags)
			} else {
				require.Empty(t, diags)
			}

			completion := state.TextDocumentCompletion(l, 1, "file:///test.hcl", tt.position)
			assert.Equal(t, tt.expected, completion)
		})
	}
}

func TestState_TextDocumentFormatting(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		document string
		expected string
	}{
		{
			name:     "empty document",
			document: "",
			expected: "",
		},
		{
			name: "unformatted locals",
			document: `locals{
foo="bar"
bar=   "baz"
}`,
			expected: `locals {
  foo = "bar"
  bar = "baz"
}`,
		},
		{
			name: "already formatted locals",
			document: `locals {
  foo = "bar"
  bar = "baz"
}`,
			expected: `locals {
  foo = "bar"
  bar = "baz"
}`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := tg.NewState()
			l := testutils.NewTestLogger(t)

			// First open the document to populate the state
			diags := state.OpenDocument(l, "file:///test.hcl", tt.document)
			require.Empty(t, diags)

			// Request formatting
			response := state.TextDocumentFormatting(l, 1, "file:///test.hcl")

			// Verify the formatting result
			require.Len(t, response.Result, 1)
			assert.Equal(t, tt.expected, response.Result[0].NewText)

			assert.Equal(t, uint32(0), response.Result[0].Range.Start.Line)
			assert.Equal(t, uint32(0), response.Result[0].Range.Start.Character)

			lines := strings.Split(tt.document, "\n")
			assert.Equal(t, uint32(len(lines)-1), response.Result[0].Range.End.Line)
			assert.Equal(t, uint32(len(lines[len(lines)-1])), response.Result[0].Range.End.Character)
		})
	}
}
