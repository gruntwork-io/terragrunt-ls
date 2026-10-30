package tg_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
)

func TestState_Definition_LocalReference_SameFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	tgPath := filepath.Join(tmpDir, "terragrunt.hcl")
	docURI := uri.File(tgPath)

	content := `locals {
  foo = "bar"
}

inputs = {
  v = local.foo
}
`
	_, err := testutils.CreateFile(tmpDir, "terragrunt.hcl", content)
	require.NoError(t, err)

	l := testutils.NewTestLogger(t)
	s := tg.NewState()
	s.OpenDocument(t.Context(), l, docURI, content)

	// Cursor on `foo` in `local.foo`.
	resp := s.Definition(l, 1, docURI, protocol.Position{Line: 5, Character: 14})

	assert.Equal(t, docURI, resp.Result.URI)
	assert.Equal(t, uint32(1), resp.Result.Range.Start.Line)
	assert.Equal(t, uint32(2), resp.Result.Range.Start.Character)
	assert.Equal(t, uint32(5), resp.Result.Range.End.Character)
}

func TestState_Definition_LocalReference_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	tgPath := filepath.Join(tmpDir, "terragrunt.hcl")
	docURI := uri.File(tgPath)

	content := `inputs = {
  v = local.nonexistent
}
`
	_, err := testutils.CreateFile(tmpDir, "terragrunt.hcl", content)
	require.NoError(t, err)

	l := testutils.NewTestLogger(t)
	s := tg.NewState()
	s.OpenDocument(t.Context(), l, docURI, content)

	// Cursor on `nonexistent` — no `locals` block defines it.
	resp := s.Definition(l, 1, docURI, protocol.Position{Line: 1, Character: 18})

	// Empty response points back at the cursor position.
	assert.Equal(t, docURI, resp.Result.URI)
	assert.Equal(t, protocol.Position{Line: 1, Character: 18}, resp.Result.Range.Start)
}

func TestState_Definition_IncludeTraversalReference(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	rootPath := filepath.Join(tmpDir, "root.hcl")
	_, err := testutils.CreateFile(tmpDir, "root.hcl", "")
	require.NoError(t, err)

	unitDir := filepath.Join(tmpDir, "app")
	require.NoError(t, os.MkdirAll(unitDir, 0o755))

	content := `include "root" {
  path = find_in_parent_folders("root.hcl")
}

inputs = {
  v = include.root.locals.region
}
`
	unitPath := filepath.Join(unitDir, "terragrunt.hcl")
	_, err = testutils.CreateFile(unitDir, "terragrunt.hcl", content)
	require.NoError(t, err)

	l := testutils.NewTestLogger(t)
	s := tg.NewState()
	s.OpenDocument(t.Context(), l, uri.File(unitPath), content)

	// Cursor on `root` in `include.root.locals.region`.
	resp := s.Definition(l, 1, uri.File(unitPath), protocol.Position{Line: 5, Character: 16})

	assert.Equal(t, uri.File(rootPath), resp.Result.URI, "should jump to root.hcl")
}
