package tg_test

import (
	"context"
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
	s.OpenDocument(context.Background(), l, docURI, content)

	// Cursor on `foo` in `local.foo`.
	resp := s.Definition(l, 1, docURI, protocol.Position{Line: 5, Character: 14})

	assert.Equal(t, docURI, resp.Result.URI)
	assert.Equal(t, uint32(1), resp.Result.Range.Start.Line)
	assert.Equal(t, uint32(2), resp.Result.Range.Start.Character)
	assert.Equal(t, uint32(5), resp.Result.Range.End.Character)
}

func TestState_Definition_LocalReference_CrossFile(t *testing.T) {
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
	s.OpenDocument(context.Background(), l, uri.File(tgPath), tgContent)

	// Cursor on `shared` in `local.shared`.
	resp := s.Definition(l, 1, uri.File(tgPath), protocol.Position{Line: 5, Character: 14})

	assert.Equal(t, uri.File(commonPath), resp.Result.URI, "should jump to common.hcl")
	assert.Equal(t, uint32(1), resp.Result.Range.Start.Line)
	assert.Equal(t, uint32(2), resp.Result.Range.Start.Character)
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
	s.OpenDocument(context.Background(), l, docURI, content)

	// Cursor on `nonexistent` — no `locals` block defines it.
	resp := s.Definition(l, 1, docURI, protocol.Position{Line: 1, Character: 18})

	// Empty response points back at the cursor position.
	assert.Equal(t, docURI, resp.Result.URI)
	assert.Equal(t, protocol.Position{Line: 1, Character: 18}, resp.Result.Range.Start)
}

func TestState_Definition_DependencyTraversalReference(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	vpcDir := filepath.Join(tmpDir, "vpc")
	require.NoError(t, os.MkdirAll(vpcDir, 0o755))
	_, err := testutils.CreateFile(vpcDir, "terragrunt.hcl", "")
	require.NoError(t, err)

	unitDir := filepath.Join(tmpDir, "app")
	require.NoError(t, os.MkdirAll(unitDir, 0o755))

	content := `dependency "vpc" {
  config_path = "../vpc"
}

inputs = {
  vpc_id = dependency.vpc.outputs.id
}
`
	unitPath := filepath.Join(unitDir, "terragrunt.hcl")
	_, err = testutils.CreateFile(unitDir, "terragrunt.hcl", content)
	require.NoError(t, err)

	l := testutils.NewTestLogger(t)
	s := tg.NewState()
	s.OpenDocument(context.Background(), l, uri.File(unitPath), content)

	// Cursor on `vpc` in `dependency.vpc.outputs.id`.
	resp := s.Definition(l, 1, uri.File(unitPath), protocol.Position{Line: 5, Character: 23})

	expectedURI := uri.File(filepath.Join(vpcDir, "terragrunt.hcl"))
	assert.Equal(t, expectedURI, resp.Result.URI, "should jump to dependent unit's terragrunt.hcl")
}
