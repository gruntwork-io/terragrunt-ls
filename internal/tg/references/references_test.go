package references_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"terragrunt-ls/internal/tg/references"
)

func TestGetReferences(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	content := `locals {
  shared = "value"
}

inputs = {
  v = local.shared
}
`
	_, err := testutils.CreateFile(tmpDir, "terragrunt.hcl", content)
	require.NoError(t, err)

	tgPath := filepath.Join(tmpDir, "terragrunt.hcl")

	l := testutils.NewTestLogger(t)
	s := tg.NewState()
	s.OpenDocument(t.Context(), l, uri.File(tgPath), content)

	t.Run("includes declaration when requested", func(t *testing.T) {
		t.Parallel()

		locs := references.GetReferences(l, s.Configs[tgPath], protocol.Position{Line: 5, Character: 14}, tgPath, true)
		require.Len(t, locs, 2, "definition + reference")

		for _, loc := range locs {
			assert.Equal(t, uri.File(tgPath), loc.URI)
		}
	})

	t.Run("excludes declaration when requested", func(t *testing.T) {
		t.Parallel()

		locs := references.GetReferences(l, s.Configs[tgPath], protocol.Position{Line: 5, Character: 14}, tgPath, false)
		require.Len(t, locs, 1, "only the reference, not the definition")

		assert.Equal(t, uri.File(tgPath), locs[0].URI)
	})

	t.Run("returns nil for non-renameable position", func(t *testing.T) {
		t.Parallel()

		locs := references.GetReferences(l, s.Configs[tgPath], protocol.Position{Line: 0, Character: 0}, tgPath, true)
		assert.Nil(t, locs)
	})
}
