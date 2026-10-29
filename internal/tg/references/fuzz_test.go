package references_test

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg/references"
	"terragrunt-ls/internal/tg/rename"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// FuzzGetReferences tests that references are exactly the rename occurrences, minus the declaration when it is excluded.
func FuzzGetReferences(f *testing.F) {
	const file = "/work/terragrunt.hcl"

	f.Add("locals {\n  env = \"dev\"\n}\n\ninputs = {\n  env = local.env\n}\n", uint32(5), uint32(14), true)
	f.Add("locals {\n  env = \"dev\"\n}\n\ninputs = {\n  env = local.env\n}\n", uint32(1), uint32(3), false)

	f.Fuzz(func(t *testing.T, document string, line, character uint32, includeDeclaration bool) {
		indexed, err := ast.ParseHCLFile(file, []byte(document))
		require.NotNil(t, indexed, "parse error: %v", err)

		st := store.Store{AST: indexed, Document: document, FileType: store.FileTypeUnit}
		position := protocol.Position{Line: line, Character: character}
		l := testutils.NewTestLogger(t)

		locations := references.GetReferences(l, st, position, file, includeDeclaration)

		target := rename.GetRenameTarget(l, st, position)

		expected := []protocol.Location{}
		for _, occurrence := range rename.FindAllOccurrences(target, file, st) {
			if occurrence.IsDefinition && !includeDeclaration {
				continue
			}

			expected = append(expected, protocol.Location{URI: uri.File(file), Range: occurrence.Range})
		}

		require.ElementsMatch(t, expected, locations)
	})
}
