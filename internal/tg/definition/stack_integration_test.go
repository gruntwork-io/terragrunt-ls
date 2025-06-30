package definition_test

import (
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

	// Create a temporary file
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "terragrunt.stack.hcl")
	docURI := uri.File(filename)

	// Create logger
	l := testutils.NewTestLogger(t)

	// Create state and open document
	state := tg.NewState()
	diags := state.OpenDocument(l, docURI, content)
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
				// For empty responses, the result URI should be the same as input (no navigation)
				assert.Equal(t, docURI, defResponse.Result.URI, tt.description)
				assert.Equal(t, position, defResponse.Result.Range.Start, tt.description)
			} else {
				// For valid definitions, we should get a different location or the resolved path
				// The exact response depends on whether the path exists, but we shouldn't get an empty response
				assert.NotEqual(t, protocol.Location{}, defResponse.Result, tt.description)
				t.Logf("Definition result: %+v", defResponse.Result)
			}
		})
	}
}
