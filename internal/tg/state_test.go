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

func TestState_TextDocumentCompletion(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		initial  string
		document string
		position protocol.Position
		expected lsp.CompletionResponse
	}{
		{
			name:     "nothing to complete",
			document: "",
			position: protocol.Position{
				Line:      0,
				Character: 0,
			},
			expected: lsp.CompletionResponse{
				Response: lsp.Response{
					RPC: "2.0",
					ID:  pointerOfInt(1),
				},
				Result: []protocol.CompletionItem{
					{
						Label: "dependency",
					},
					{
						Label: "inputs",
					},
					{
						Label: "local",
					},
					{
						Label: "locals",
					},
					{
						Label: "feature",
					},
					{
						Label: "terraform",
					},
					{
						Label: "remote_state",
					},
					{
						Label: "include",
					},
					{
						Label: "dependencies",
					},
					{
						Label: "generate",
					},
					{
						Label: "engine",
					},
					{
						Label: "exclude",
					},
					{
						Label: "download_dir",
					},
					{
						Label: "prevent_destroy",
					},
					{
						Label: "iam_role",
					},
					{
						Label: "iam_assume_role_duration",
					},
					{
						Label: "iam_assume_role_session_name",
					},
					{
						Label: "iam_web_identity_token",
					},
					{
						Label: "terraform_binary",
					},
					{
						Label: "terraform_version_constraint",
					},
					{
						Label: "terragrunt_version_constraint",
					},
				},
			},
		},
		// TODO: Fix this test as the next feature.
		//
		//		{
		//			name: "empty local",
		//			initial: `locals {
		//	foo = "bar"
		// }`,
		//
		//			document: `locals {
		//	foo = "bar"
		//	bar = local.
		// }`,
		//			position: protocol.Position{
		//				Line:      2,
		//				Character: 12,
		//			},
		//			expected: lsp.CompletionResponse{
		//				Response: lsp.Response{
		//					RPC: "2.0",
		//					ID:  pointerOfInt(1),
		//				},
		//				Result: []protocol.CompletionItem{
		//					Label: "local.foo",
		//				},
		//			},
		//		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := tg.NewState()

			l := newTestLogger(t)

			diags := state.OpenDocument(l, "file:///foo/bar.hcl", tt.initial)
			require.Empty(t, diags)

			_ = state.UpdateDocument(l, "file:///foo/bar.hcl", tt.document)

			require.Len(t, state.Configs, 1)

			completion := state.TextDocumentCompletion(l, 1, "file:///foo/bar.hcl", tt.position)
			assert.Equal(t, tt.expected, completion)
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
