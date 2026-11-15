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

const (
	unitIncludingRoot = `include "root" {
  path = find_in_parent_folders("root.hcl")
}
`

	rootWithUnknownFunction = `inputs = {
  name = not_a_function()
}
`
)

// sharedFixture is a root.hcl and the terragrunt.hcl of a unit below it.
type sharedFixture struct {
	v       *venv.Venv
	rootURI uri.URI
	unitURI uri.URI
}

func newSharedFixture(t *testing.T) sharedFixture {
	t.Helper()

	v := venv.MemVenv()
	tmpDir := testutils.WorkDir(t, v)

	_, err := testutils.CreateFile(v, tmpDir, "root.hcl", "")
	require.NoError(t, err)

	unitDir := filepath.Join(tmpDir, "unit")
	require.NoError(t, v.FS.MkdirAll(unitDir, 0755))

	return sharedFixture{
		v:       v,
		rootURI: uri.File(filepath.Join(tmpDir, "root.hcl")),
		unitURI: uri.File(filepath.Join(unitDir, "terragrunt.hcl")),
	}
}

// TestState_OpenDocument_IncludedFile_ReportsDiagnostics pins that a file gets diagnostics once an open unit includes
// it.
func TestState_OpenDocument_IncludedFile_ReportsDiagnostics(t *testing.T) {
	t.Parallel()

	f := newSharedFixture(t)
	l := testutils.NewTestLogger(t)
	s := tg.NewState(testutils.NewDocs(t))

	require.Empty(t, s.OpenDocument(t.Context(), l, f.v, venv.Policy{}, f.unitURI, unitIncludingRoot))

	diags := s.OpenDocument(t.Context(), l, f.v, venv.Policy{}, f.rootURI, rootWithUnknownFunction)

	assert.NotEmpty(t, diags)
}

// TestState_OpenDocument_FileNoUnitIncludes_ReportsNoDiagnostics pins that a file no open unit includes gets no
// diagnostics and no completions.
func TestState_OpenDocument_FileNoUnitIncludes_ReportsNoDiagnostics(t *testing.T) {
	t.Parallel()

	f := newSharedFixture(t)
	l := testutils.NewTestLogger(t)
	s := tg.NewState(testutils.NewDocs(t))

	diags := s.OpenDocument(t.Context(), l, f.v, venv.Policy{}, f.rootURI, rootWithUnknownFunction)

	assert.Empty(t, diags)
	assert.Empty(t, s.TextDocumentCompletion(l, 1, f.rootURI, protocol.Position{}).Result)
}

// TestState_UpdateDocument_UnitDropsInclude_ReportsNoDiagnostics pins that a file loses its diagnostics when the only
// unit that included it stops doing so.
func TestState_UpdateDocument_UnitDropsInclude_ReportsNoDiagnostics(t *testing.T) {
	t.Parallel()

	f := newSharedFixture(t)
	l := testutils.NewTestLogger(t)
	s := tg.NewState(testutils.NewDocs(t))

	s.OpenDocument(t.Context(), l, f.v, venv.Policy{}, f.unitURI, unitIncludingRoot)
	s.UpdateDocument(t.Context(), l, f.v, venv.Policy{}, f.unitURI, "inputs = {}\n")

	diags := s.OpenDocument(t.Context(), l, f.v, venv.Policy{}, f.rootURI, rootWithUnknownFunction)

	assert.Empty(t, diags)
}

// TestState_UpdateDocument_UnitFailsToParse_KeepsInclude pins that a file keeps its diagnostics while the unit that
// includes it has a syntax error.
func TestState_UpdateDocument_UnitFailsToParse_KeepsInclude(t *testing.T) {
	t.Parallel()

	f := newSharedFixture(t)
	l := testutils.NewTestLogger(t)
	s := tg.NewState(testutils.NewDocs(t))

	s.OpenDocument(t.Context(), l, f.v, venv.Policy{}, f.unitURI, unitIncludingRoot)
	s.UpdateDocument(t.Context(), l, f.v, venv.Policy{}, f.unitURI, `include "root" {`)

	diags := s.OpenDocument(t.Context(), l, f.v, venv.Policy{}, f.rootURI, rootWithUnknownFunction)

	assert.NotEmpty(t, diags)
}

// TestState_Hover_IncludedFile pins that hovering a local in an included file shows its value.
func TestState_Hover_IncludedFile(t *testing.T) {
	t.Parallel()

	f := newSharedFixture(t)
	l := testutils.NewTestLogger(t)
	s := tg.NewState(testutils.NewDocs(t))

	const root = `locals {
  region = "us-east-1"
  name   = local.region
}
`

	s.OpenDocument(t.Context(), l, f.v, venv.Policy{}, f.unitURI, unitIncludingRoot)
	require.Empty(t, s.OpenDocument(t.Context(), l, f.v, venv.Policy{}, f.rootURI, root))

	resp := s.Hover(t.Context(), l, 1, f.rootURI, protocol.Position{Line: 2, Character: 14})

	require.NotNil(t, resp.Result)
	assert.Equal(t, "```hcl\nregion = \"us-east-1\"\n```", resp.Result.Contents.Value)
}

// TestState_TextDocumentCompletion_IncludedFile pins that an included file gets completions.
func TestState_TextDocumentCompletion_IncludedFile(t *testing.T) {
	t.Parallel()

	f := newSharedFixture(t)
	l := testutils.NewTestLogger(t)

	s := tg.NewState(testutils.NewDocs(t))
	s.OpenDocument(t.Context(), l, f.v, venv.Policy{}, f.unitURI, unitIncludingRoot)
	s.OpenDocument(t.Context(), l, f.v, venv.Policy{}, f.rootURI, "")

	items := s.TextDocumentCompletion(l, 1, f.rootURI, protocol.Position{}).Result

	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, item.Label)
	}

	assert.Contains(t, labels, "remote_state")
}
