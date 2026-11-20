package tg_test

import (
	"testing"

	"github.com/gruntwork-io/terragrunt/pkg/venv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
)

// Each 🧪 is one HCL column but two UTF-16 code units, so a position that HCL and LSP count
// the same way lands on the wrong character after it.
const utf16Doc = `locals {
  /* 🧪 */ foo = "bar"
}

inputs = {
  v = ["🧪🧪", local.foo]
  w = ["🧪🧪", get_env("x")]
}
`

// Positions in utf16Doc, in UTF-16 code units.
var (
	fooDeclaration = protocol.Range{
		Start: protocol.Position{Line: 1, Character: 11},
		End:   protocol.Position{Line: 1, Character: 14},
	}
	fooReference = protocol.Range{
		Start: protocol.Position{Line: 5, Character: 21},
		End:   protocol.Position{Line: 5, Character: 24},
	}
	lastLetterOfGetEnv = protocol.Position{Line: 6, Character: 21}
)

func TestState_UTF16Positions(t *testing.T) {
	t.Parallel()

	const docURI = uri.URI("file:///terragrunt.hcl")

	open := func(t *testing.T, doc string) (tg.State, []protocol.Diagnostic) {
		t.Helper()

		s := tg.NewState(testutils.NewDocs(t))
		diags := s.OpenDocument(t.Context(), testutils.NewTestLogger(t), venv.MemVenv(), venv.Policy{}, docURI, doc)

		return s, diags
	}

	t.Run("definition of a local", func(t *testing.T) {
		t.Parallel()

		s, _ := open(t, utf16Doc)

		resp := s.Definition(testutils.NewTestLogger(t), venv.MemVenv(), 1, docURI, fooReference.Start)

		assert.Equal(t, protocol.Location{URI: docURI, Range: fooDeclaration}, resp.Result)
	})

	t.Run("rename of a local", func(t *testing.T) {
		t.Parallel()

		s, _ := open(t, utf16Doc)

		resp := s.TextDocumentRename(testutils.NewTestLogger(t), 1, docURI, fooReference.Start, "renamed")
		require.NotNil(t, resp.Result)

		assert.Equal(t, []protocol.TextEdit{
			{Range: fooDeclaration, NewText: "renamed"},
			{Range: fooReference, NewText: "renamed"},
		}, resp.Result.Changes[docURI])
	})

	t.Run("hover on a function", func(t *testing.T) {
		t.Parallel()

		s, _ := open(t, utf16Doc)

		resp := s.Hover(t.Context(), testutils.NewTestLogger(t), 1, docURI, lastLetterOfGetEnv)
		require.NotNil(t, resp.Result)

		assert.Contains(t, resp.Result.Contents.Value, "functions/get_env")
	})

	t.Run("diagnostic range", func(t *testing.T) {
		t.Parallel()

		_, diags := open(t, "inputs = {\n  v = [\"🧪🧪\", nope]\n}\n")
		require.Len(t, diags, 1)

		assert.Equal(t, protocol.Range{
			Start: protocol.Position{Line: 1, Character: 15},
			End:   protocol.Position{Line: 1, Character: 19},
		}, diags[0].Range)
	})

	t.Run("formatting replaces through the end of the document", func(t *testing.T) {
		t.Parallel()

		s, _ := open(t, "locals{\nfoo=\"🧪\"}")

		resp := s.TextDocumentFormatting(testutils.NewTestLogger(t), 1, docURI)
		require.Len(t, resp.Result, 1)

		assert.Equal(t, protocol.Position{Line: 1, Character: 9}, resp.Result[0].Range.End)
	})
}
