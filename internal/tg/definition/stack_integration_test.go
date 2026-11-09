package definition_test

import (
	"os"
	"path/filepath"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestStackDefinition_Integration(t *testing.T) {
	t.Parallel()

	content := `unit "database" {
  source = "./database"
  path   = "db"
}

stack "nested" {
  source = "./nested-stack"
  path   = "nested"
}`

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "terragrunt.stack.hcl")
	docURI := uri.File(filename)

	l := testutils.NewTestLogger(t)

	state := tg.NewState()
	diags := state.OpenDocument(t.Context(), l, docURI, content)
	require.Empty(t, diags, "Expected no diagnostics")

	tests := []struct {
		name        string
		description string
		line        uint32
		character   uint32
		expectEmpty bool
	}{
		{
			name:        "unit source attribute value",
			line:        1,  // second line (0-indexed)
			character:   15, // middle of "./database"
			expectEmpty: false,
			description: "clicking on unit source attribute value should provide definition",
		},
		{
			name:        "stack source attribute value",
			line:        5,  // sixth line (0-indexed)
			character:   15, // middle of "./nested-stack"
			expectEmpty: false,
			description: "clicking on stack source attribute value should provide definition",
		},
		{
			name:        "path attribute value",
			line:        2,  // third line (0-indexed)
			character:   10, // middle of "db"
			expectEmpty: false,
			description: "clicking on path attribute value should provide definition",
		},
		{
			name:        "outside attributes",
			line:        0, // first line
			character:   5, // middle of "unit"
			expectEmpty: true,
			description: "clicking outside of source/path attributes should not provide definition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			position := protocol.Position{Line: tt.line, Character: tt.character}
			defResponse := state.Definition(l, 1, docURI, position)

			if tt.expectEmpty {
				assert.Equal(t, docURI, defResponse.Result.URI, tt.description)
				assert.Equal(t, position, defResponse.Result.Range.Start, tt.description)
			} else {
				assert.NotEqual(t, protocol.Location{}, defResponse.Result, tt.description)
				t.Logf("Definition result: %+v", defResponse.Result)
			}
		})
	}
}

func TestStackDefinition_PathOnlyOnPathAttribute(t *testing.T) {
	t.Parallel()

	content := `unit "database" {
  source = "./database"
  path   = "db"
}`

	tmpDir := t.TempDir()
	generated := filepath.Join(tmpDir, ".terragrunt-stack", "db", "terragrunt.hcl")
	require.NoError(t, os.MkdirAll(filepath.Dir(generated), 0755))
	require.NoError(t, os.WriteFile(generated, []byte(""), 0644))

	docURI := uri.File(filepath.Join(tmpDir, "terragrunt.stack.hcl"))
	l := testutils.NewTestLogger(t)

	state := tg.NewState()
	require.Empty(t, state.OpenDocument(t.Context(), l, docURI, content))

	onPath := state.Definition(l, 1, docURI, protocol.Position{Line: 2, Character: 10})
	assert.Equal(t, uri.File(generated), onPath.Result.URI)

	// On the `unit` keyword, do not navigate, even though the generated file exists.
	onKeyword := state.Definition(l, 1, docURI, protocol.Position{Line: 0, Character: 2})
	assert.Equal(t, docURI, onKeyword.Result.URI)
}

func TestStackDefinition_ComputedPath(t *testing.T) {
	t.Parallel()

	content := `locals {
  name = "db"
}

unit "database" {
  source = "./database"
  path   = local.name
}`

	tmpDir := t.TempDir()
	generated := filepath.Join(tmpDir, ".terragrunt-stack", "db", "terragrunt.hcl")
	require.NoError(t, os.MkdirAll(filepath.Dir(generated), 0755))
	require.NoError(t, os.WriteFile(generated, []byte(""), 0644))

	docURI := uri.File(filepath.Join(tmpDir, "terragrunt.stack.hcl"))
	l := testutils.NewTestLogger(t)

	state := tg.NewState()
	require.Empty(t, state.OpenDocument(t.Context(), l, docURI, content))

	resp := state.Definition(l, 1, docURI, protocol.Position{Line: 6, Character: 14})
	assert.Equal(t, uri.File(generated), resp.Result.URI)
}
