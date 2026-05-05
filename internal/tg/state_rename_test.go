package tg_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
)

func TestState_PrepareRename(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name      string
		document  string
		wantPlace string
		position  protocol.Position
		wantStart protocol.Position
		wantEnd   protocol.Position
		wantNil   bool
	}{
		{
			name: "local definition",
			document: `locals {
  foo = "bar"
}`,
			position:  protocol.Position{Line: 1, Character: 3},
			wantPlace: "foo",
			wantStart: protocol.Position{Line: 1, Character: 2},
			wantEnd:   protocol.Position{Line: 1, Character: 5},
		},
		{
			name: "local reference",
			document: `locals { foo = "bar" }
inputs = { v = local.foo }`,
			position:  protocol.Position{Line: 1, Character: 23},
			wantPlace: "foo",
			wantStart: protocol.Position{Line: 1, Character: 21},
			wantEnd:   protocol.Position{Line: 1, Character: 24},
		},
		{
			name: "non-renameable position returns nil",
			document: `locals {
  foo = "bar"
}`,
			position: protocol.Position{Line: 0, Character: 0},
			wantNil:  true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			tgPath := filepath.Join(tmpDir, "terragrunt.hcl")
			docURI := uri.File(tgPath)

			l := testutils.NewTestLogger(t)
			s := tg.NewState()
			s.OpenDocument(context.Background(), l, docURI, tt.document)

			resp := s.PrepareRename(l, 1, docURI, tt.position)

			if tt.wantNil {
				assert.Nil(t, resp.Result)
				return
			}

			require.NotNil(t, resp.Result)
			assert.Equal(t, tt.wantPlace, resp.Result.Placeholder)
			assert.Equal(t, tt.wantStart, resp.Result.Range.Start)
			assert.Equal(t, tt.wantEnd, resp.Result.Range.End)
		})
	}
}

func TestState_TextDocumentRename(t *testing.T) {
	t.Parallel()

	t.Run("rejects invalid identifier", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tgPath := filepath.Join(tmpDir, "terragrunt.hcl")
		docURI := uri.File(tgPath)

		l := testutils.NewTestLogger(t)
		s := tg.NewState()
		s.OpenDocument(context.Background(), l, docURI, `locals { foo = "bar" }`)

		resp := s.TextDocumentRename(l, 1, docURI, protocol.Position{Line: 0, Character: 9}, "1invalid")
		assert.Nil(t, resp.Result)
	})

	t.Run("renames local across same-folder files", func(t *testing.T) {
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

		resp := s.TextDocumentRename(l, 1, uri.File(tgPath), protocol.Position{Line: 5, Character: 14}, "renamed")
		require.NotNil(t, resp.Result)
		require.NotNil(t, resp.Result.Changes)

		assert.Len(t, resp.Result.Changes, 2, "edits should span both files")

		commonEdits := resp.Result.Changes[uri.File(commonPath)]
		require.Len(t, commonEdits, 1)
		assert.Equal(t, "renamed", commonEdits[0].NewText)

		tgEdits := resp.Result.Changes[uri.File(tgPath)]
		require.Len(t, tgEdits, 1)
		assert.Equal(t, "renamed", tgEdits[0].NewText)
	})

	t.Run("returns nil for non-renameable position", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tgPath := filepath.Join(tmpDir, "terragrunt.hcl")
		docURI := uri.File(tgPath)

		l := testutils.NewTestLogger(t)
		s := tg.NewState()
		s.OpenDocument(context.Background(), l, docURI, `locals { foo = "bar" }`)

		resp := s.TextDocumentRename(l, 1, docURI, protocol.Position{Line: 0, Character: 0}, "valid")
		assert.Nil(t, resp.Result)
	})

	t.Run("works on auxiliary HCL files (FileTypeUnknown)", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		commonPath := filepath.Join(tmpDir, "common.hcl")
		docURI := uri.File(commonPath)

		l := testutils.NewTestLogger(t)
		s := tg.NewState()
		s.OpenDocument(context.Background(), l, docURI, `locals { foo = "bar" }`)

		resp := s.TextDocumentRename(l, 1, docURI, protocol.Position{Line: 0, Character: 9}, "renamed")
		require.NotNil(t, resp.Result)

		edits := resp.Result.Changes[docURI]
		require.Len(t, edits, 1)
		assert.Equal(t, "renamed", edits[0].NewText)
	})
}
