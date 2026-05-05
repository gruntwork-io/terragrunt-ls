package rename_test

import (
	"path/filepath"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"terragrunt-ls/internal/tg/rename"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestIsValidIdentifier(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name string
		want bool
	}{
		{"foo", true},
		{"_bar", true},
		{"my_local", true},
		{"a1", true},
		{"my-include", true},
		{"", false},
		{"1foo", false},
		{"foo bar", false},
		{"foo.bar", false},
		{"local.foo", false},
		{"foo!", false},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, rename.IsValidIdentifier(tt.name))
		})
	}
}

func TestGetRenameTarget(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name            string
		document        string
		expectedName    string
		expectedContext string
		position        protocol.Position
	}{
		{
			name:            "empty document",
			document:        "",
			position:        protocol.Position{Line: 0, Character: 0},
			expectedContext: rename.RenameContextNull,
		},
		{
			name: "cursor on local definition",
			document: `locals {
  foo = "bar"
}`,
			position:        protocol.Position{Line: 1, Character: 3},
			expectedName:    "foo",
			expectedContext: rename.RenameContextLocal,
		},
		{
			name: "cursor on local reference",
			document: `locals {
  foo = "bar"
}
inputs = {
  v = local.foo
}`,
			position:        protocol.Position{Line: 4, Character: 14},
			expectedName:    "foo",
			expectedContext: rename.RenameContextLocal,
		},
		{
			name:            "cursor on local keyword in reference",
			document:        `inputs = { v = local.foo }`,
			position:        protocol.Position{Line: 0, Character: 16},
			expectedName:    "foo",
			expectedContext: rename.RenameContextLocal,
		},
		{
			name: "cursor on unrelated traversal root",
			document: `inputs = {
  v = path.module
}`,
			position:        protocol.Position{Line: 1, Character: 8},
			expectedContext: rename.RenameContextNull,
		},
		{
			name: "cursor on locals block keyword",
			document: `locals {
  foo = "bar"
}`,
			position:        protocol.Position{Line: 0, Character: 3},
			expectedContext: rename.RenameContextNull,
		},
		{
			name: "cursor on whitespace",
			document: `locals {
  foo = "bar"
}`,
			position:        protocol.Position{Line: 1, Character: 0},
			expectedContext: rename.RenameContextNull,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewTestLogger(t)
			s := tg.NewState()

			docURI := uri.File("/test/terragrunt.hcl")
			s.OpenDocument(t.Context(), l, docURI, tt.document)

			target := rename.GetRenameTarget(l, s.Configs[docURI.Filename()], tt.position)

			assert.Equal(t, tt.expectedContext, target.Context)
			if tt.expectedContext != rename.RenameContextNull {
				assert.Equal(t, tt.expectedName, target.Name)
			}
		})
	}
}

func TestFindAllOccurrences_Local(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	hclPath := filepath.Join(tmpDir, "terragrunt.hcl")

	content := `locals {
  foo = "bar"
}

dependency "a" {
  config_path = local.foo
}

dependency "b" {
  config_path = local.foo
}
`
	_, err := testutils.CreateFile(tmpDir, "terragrunt.hcl", content)
	require.NoError(t, err)

	l := testutils.NewTestLogger(t)
	s := tg.NewState()
	docURI := uri.File(hclPath)
	s.OpenDocument(t.Context(), l, docURI, content)

	st := s.Configs[hclPath]

	target := rename.GetRenameTarget(l, st, protocol.Position{Line: 1, Character: 3})
	require.Equal(t, rename.RenameContextLocal, target.Context)
	require.Equal(t, "foo", target.Name)

	occs := rename.FindAllOccurrences(target, hclPath, st)
	require.Len(t, occs, 3)

	var defs int

	for _, occ := range occs {
		assert.Equal(t, hclPath, occ.File)

		if occ.IsDefinition {
			defs++
		}
	}

	assert.Equal(t, 1, defs, "exactly one definition occurrence")
}

func TestFindAllOccurrences_NoDefinition(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	hclPath := filepath.Join(tmpDir, "terragrunt.hcl")

	// References to a local that has no declaration in this file.
	content := `inputs = {
  v = local.shared
}
`
	_, err := testutils.CreateFile(tmpDir, "terragrunt.hcl", content)
	require.NoError(t, err)

	l := testutils.NewTestLogger(t)
	s := tg.NewState()
	docURI := uri.File(hclPath)
	s.OpenDocument(t.Context(), l, docURI, content)

	st := s.Configs[hclPath]

	target := rename.GetRenameTarget(l, st, protocol.Position{Line: 1, Character: 14})
	require.Equal(t, rename.RenameContextLocal, target.Context)
	require.Equal(t, "shared", target.Name)

	occs := rename.FindAllOccurrences(target, hclPath, st)
	require.Len(t, occs, 1, "only the reference, no declaration in this file")
	assert.False(t, occs[0].IsDefinition)
}
