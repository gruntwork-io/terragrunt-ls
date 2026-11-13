package tg_test

import (
	"path/filepath"
	"testing"

	"github.com/gruntwork-io/terragrunt/pkg/venv"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
)

func TestState_Definition_LocalReference_SameFile(t *testing.T) {
	t.Parallel()

	v := venv.MemVenv()

	tmpDir := testutils.WorkDir(t, v)
	tgPath := filepath.Join(tmpDir, "terragrunt.hcl")
	docURI := uri.File(tgPath)

	content := `locals {
  foo = "bar"
}

inputs = {
  v = local.foo
}
`
	_, err := testutils.CreateFile(v, tmpDir, "terragrunt.hcl", content)
	require.NoError(t, err)

	l := testutils.NewTestLogger(t)
	s := tg.NewState(testutils.NewDocs(t))
	s.OpenDocument(t.Context(), l, v, venv.Policy{}, docURI, content)

	// Cursor on `foo` in `local.foo`.
	resp := s.Definition(l, v, 1, docURI, protocol.Position{Line: 5, Character: 14})

	assert.Equal(t, docURI, resp.Result.URI)
	assert.Equal(t, uint32(1), resp.Result.Range.Start.Line)
	assert.Equal(t, uint32(2), resp.Result.Range.Start.Character)
	assert.Equal(t, uint32(5), resp.Result.Range.End.Character)
}

func TestState_Definition_LocalReference_NotFound(t *testing.T) {
	t.Parallel()

	v := venv.MemVenv()

	tmpDir := testutils.WorkDir(t, v)
	tgPath := filepath.Join(tmpDir, "terragrunt.hcl")
	docURI := uri.File(tgPath)

	content := `inputs = {
  v = local.nonexistent
}
`
	_, err := testutils.CreateFile(v, tmpDir, "terragrunt.hcl", content)
	require.NoError(t, err)

	l := testutils.NewTestLogger(t)
	s := tg.NewState(testutils.NewDocs(t))
	s.OpenDocument(t.Context(), l, v, venv.Policy{}, docURI, content)

	// Cursor on `nonexistent` — no `locals` block defines it.
	resp := s.Definition(l, v, 1, docURI, protocol.Position{Line: 1, Character: 18})

	// Empty response points back at the cursor position.
	assert.Equal(t, docURI, resp.Result.URI)
	assert.Equal(t, protocol.Position{Line: 1, Character: 18}, resp.Result.Range.Start)
}
