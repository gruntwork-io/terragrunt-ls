package rename_test

import (
	"context"
	"path/filepath"
	"sort"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"terragrunt-ls/internal/tg/rename"
	"terragrunt-ls/internal/tg/store"
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
			name: "cursor on include label",
			document: `include "root" {
  path = "root.hcl"
}`,
			position:        protocol.Position{Line: 0, Character: 10},
			expectedName:    "root",
			expectedContext: rename.RenameContextInclude,
		},
		{
			name: "cursor on dependency label",
			document: `dependency "vpc" {
  config_path = "../vpc"
}`,
			position:        protocol.Position{Line: 0, Character: 13},
			expectedName:    "vpc",
			expectedContext: rename.RenameContextDependency,
		},
		{
			name: "cursor on dependency outputs reference",
			document: `inputs = {
  id = dependency.vpc.outputs.id
}`,
			position:        protocol.Position{Line: 1, Character: 19},
			expectedName:    "vpc",
			expectedContext: rename.RenameContextDependency,
		},
		{
			name: "cursor on outputs step is not renameable",
			document: `inputs = {
  id = dependency.vpc.outputs.id
}`,
			position:        protocol.Position{Line: 1, Character: 24},
			expectedContext: rename.RenameContextNull,
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
			s.OpenDocument(context.Background(), l, docURI, tt.document)

			target := rename.GetRenameTarget(l, s.Configs[docURI.Filename()], tt.position)

			assert.Equal(t, tt.expectedContext, target.Context)
			if tt.expectedContext != rename.RenameContextNull {
				assert.Equal(t, tt.expectedName, target.Name)
			}
		})
	}
}

func TestFindAllOccurrences_LocalSingleFile(t *testing.T) {
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
	s.OpenDocument(context.Background(), l, docURI, content)

	target := rename.GetRenameTarget(l, s.Configs[hclPath], protocol.Position{Line: 1, Character: 3})
	require.Equal(t, rename.RenameContextLocal, target.Context)
	require.Equal(t, "foo", target.Name)

	occs := rename.FindAllOccurrences(l, target, hclPath, s.Configs)
	require.Len(t, occs, 3)

	for _, occ := range occs {
		assert.Equal(t, hclPath, occ.File)
	}
}

func TestFindAllOccurrences_LocalCrossFile(t *testing.T) {
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

	// Open only terragrunt.hcl in the editor; common.hcl stays on disk only.
	s.OpenDocument(context.Background(), l, uri.File(tgPath), tgContent)

	// Cursor on the local.shared reference.
	target := rename.GetRenameTarget(l, s.Configs[tgPath], protocol.Position{Line: 5, Character: 14})
	require.Equal(t, rename.RenameContextLocal, target.Context)
	require.Equal(t, "shared", target.Name)

	occs := rename.FindAllOccurrences(l, target, tgPath, s.Configs)
	require.Len(t, occs, 2)

	files := []string{occs[0].File, occs[1].File}
	sort.Strings(files)
	assert.Equal(t, []string{commonPath, tgPath}, files)
}

func TestFindAllOccurrences_SkipsStackAndValues(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	mainContent := `locals { foo = "v" }
inputs = { v = local.foo }
`
	_, err := testutils.CreateFile(tmpDir, "terragrunt.hcl", mainContent)
	require.NoError(t, err)

	// These would parse but should not be scanned.
	_, err = testutils.CreateFile(tmpDir, "terragrunt.stack.hcl", `unit "x" { source = "./x" path = "x" }`)
	require.NoError(t, err)
	_, err = testutils.CreateFile(tmpDir, "terragrunt.values.hcl", `foo = "shadow"`)
	require.NoError(t, err)

	tgPath := filepath.Join(tmpDir, "terragrunt.hcl")

	l := testutils.NewTestLogger(t)
	s := tg.NewState()
	s.OpenDocument(context.Background(), l, uri.File(tgPath), mainContent)

	target := rename.GetRenameTarget(l, s.Configs[tgPath], protocol.Position{Line: 0, Character: 9})
	require.Equal(t, rename.RenameContextLocal, target.Context)

	occs := rename.FindAllOccurrences(l, target, tgPath, s.Configs)
	for _, occ := range occs {
		assert.Equal(t, "terragrunt.hcl", filepath.Base(occ.File), "must not include stack/values files")
	}
}

func TestFindAllOccurrences_PrefersInMemoryAST(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// On disk: defines a different name.
	diskContent := `locals { other = "v" }`
	_, err := testutils.CreateFile(tmpDir, "common.hcl", diskContent)
	require.NoError(t, err)

	tgContent := `inputs = { v = local.shared }`
	_, err = testutils.CreateFile(tmpDir, "terragrunt.hcl", tgContent)
	require.NoError(t, err)

	tgPath := filepath.Join(tmpDir, "terragrunt.hcl")
	commonPath := filepath.Join(tmpDir, "common.hcl")

	l := testutils.NewTestLogger(t)
	s := tg.NewState()

	// Open both files; for common.hcl provide an in-memory version that
	// matches the local name `shared`.
	s.OpenDocument(context.Background(), l, uri.File(tgPath), tgContent)
	s.OpenDocument(context.Background(), l, uri.File(commonPath), `locals { shared = "v" }`)

	target := rename.GetRenameTarget(l, s.Configs[tgPath], protocol.Position{Line: 0, Character: 22})
	require.Equal(t, rename.RenameContextLocal, target.Context)

	configs := map[string]store.Store{
		tgPath:     s.Configs[tgPath],
		commonPath: s.Configs[commonPath],
	}

	occs := rename.FindAllOccurrences(l, target, tgPath, configs)
	require.Len(t, occs, 2, "definition (from in-memory common) + reference")
}

func TestFindAllOccurrences_DependencyLabel(t *testing.T) {
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
	s.OpenDocument(context.Background(), l, uri.File(tgPath), content)

	// Cursor on the dependency label "vpc".
	target := rename.GetRenameTarget(l, s.Configs[tgPath], protocol.Position{Line: 0, Character: 13})
	require.Equal(t, rename.RenameContextDependency, target.Context)
	require.Equal(t, "vpc", target.Name)

	occs := rename.FindAllOccurrences(l, target, tgPath, s.Configs)
	require.Len(t, occs, 2, "definition label + outputs reference")
}

func TestFindAllOccurrences_IncludeLabel(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	content := `include "root" {
  path = "root.hcl"
}

inputs = {
  region = include.root.locals.region
}
`
	_, err := testutils.CreateFile(tmpDir, "terragrunt.hcl", content)
	require.NoError(t, err)

	tgPath := filepath.Join(tmpDir, "terragrunt.hcl")

	l := testutils.NewTestLogger(t)
	s := tg.NewState()
	s.OpenDocument(context.Background(), l, uri.File(tgPath), content)

	target := rename.GetRenameTarget(l, s.Configs[tgPath], protocol.Position{Line: 0, Character: 10})
	require.Equal(t, rename.RenameContextInclude, target.Context)
	require.Equal(t, "root", target.Name)

	occs := rename.FindAllOccurrences(l, target, tgPath, s.Configs)
	require.Len(t, occs, 2, "definition label + include.root.* reference")
}
