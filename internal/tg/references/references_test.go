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

	commonContent := `locals {
  shared = "value"
}
`
	_, err := testutils.CreateFile(tmpDir, "common.hcl", commonContent)
	require.NoError(t, err)

	tgContent := `include "common" {
  path = "common.hcl"
}

inputs = {
  v = local.shared
}
`
	_, err = testutils.CreateFile(tmpDir, "terragrunt.hcl", tgContent)
	require.NoError(t, err)

	tgPath := filepath.Join(tmpDir, "terragrunt.hcl")
	commonPath := filepath.Join(tmpDir, "common.hcl")

	l := testutils.NewTestLogger(t)
	s := tg.NewState()
	s.OpenDocument(t.Context(), l, uri.File(tgPath), tgContent)

	t.Run("includes declaration when requested", func(t *testing.T) {
		t.Parallel()

		locs := references.GetReferences(l, s.Configs[tgPath], protocol.Position{Line: 5, Character: 14}, tgPath, s.Configs, true)
		require.Len(t, locs, 2, "definition + reference")

		// Sorted by file name then position; common.hcl comes first alphabetically.
		assert.Equal(t, uri.File(commonPath), locs[0].URI)
		assert.Equal(t, uri.File(tgPath), locs[1].URI)
	})

	t.Run("excludes declaration when requested", func(t *testing.T) {
		t.Parallel()

		locs := references.GetReferences(l, s.Configs[tgPath], protocol.Position{Line: 5, Character: 14}, tgPath, s.Configs, false)
		require.Len(t, locs, 1, "only the reference, not the definition")

		assert.Equal(t, uri.File(tgPath), locs[0].URI)
	})

	t.Run("returns nil for non-renameable position", func(t *testing.T) {
		t.Parallel()

		locs := references.GetReferences(l, s.Configs[tgPath], protocol.Position{Line: 0, Character: 0}, tgPath, s.Configs, true)
		assert.Nil(t, locs)
	})
}

func TestGetReferences_DependencyLabel(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	content := `dependency "vpc" {
  config_path = "../vpc"
}

inputs = {
  vpc_id = dependency.vpc.outputs.id
}
`
	_, err := testutils.CreateFile(tmpDir, "terragrunt.hcl", content)
	require.NoError(t, err)

	tgPath := filepath.Join(tmpDir, "terragrunt.hcl")
	l := testutils.NewTestLogger(t)
	s := tg.NewState()
	s.OpenDocument(t.Context(), l, uri.File(tgPath), content)

	// Cursor on the reference (`vpc` in dependency.vpc.outputs.id).
	locs := references.GetReferences(l, s.Configs[tgPath], protocol.Position{Line: 5, Character: 23}, tgPath, s.Configs, true)
	require.Len(t, locs, 2, "label definition + outputs reference")
}
