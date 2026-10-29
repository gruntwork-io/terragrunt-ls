package rename_test

import (
	"cmp"
	"slices"
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg/rename"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

// FuzzFindAllOccurrences tests that a rename target is always one of its own occurrences and that occurrences come back sorted.
func FuzzFindAllOccurrences(f *testing.F) {
	const file = "/work/terragrunt.hcl"

	f.Add("locals {\n  env  = \"dev\"\n  name = \"${local.env}-app\"\n}\n\ninputs = {\n  env = local.env\n}\n", uint32(1), uint32(3))
	f.Add("locals {\n  env  = \"dev\"\n  name = \"${local.env}-app\"\n}\n\ninputs = {\n  env = local.env\n}\n", uint32(6), uint32(14))
	f.Add("locals {\n  a = local.b\n  b = local.a\n}\n", uint32(1), uint32(13))

	f.Fuzz(func(t *testing.T, document string, line, character uint32) {
		indexed, err := ast.ParseHCLFile(file, []byte(document))
		require.NotNil(t, indexed, "parse error: %v", err)

		st := store.Store{AST: indexed, Document: document, FileType: store.FileTypeUnit}
		position := protocol.Position{Line: line, Character: character}

		target := rename.GetRenameTarget(testutils.NewTestLogger(t), st, position)
		occurrences := rename.FindAllOccurrences(target, file, st)

		if target.Context == rename.RenameContextNull {
			require.Empty(t, occurrences)

			return
		}

		require.Equal(t, rename.RenameContextLocal, target.Context)
		require.NotEmpty(t, target.Name)

		ranges := make([]protocol.Range, 0, len(occurrences))
		for _, occurrence := range occurrences {
			require.Equal(t, file, occurrence.File)

			ranges = append(ranges, occurrence.Range)
		}

		require.Contains(t, ranges, target.IdentRange)
		require.True(t, slices.IsSortedFunc(ranges, compareStart), "occurrences are not sorted")
	})
}

func compareStart(a, b protocol.Range) int {
	return cmp.Or(cmp.Compare(a.Start.Line, b.Start.Line), cmp.Compare(a.Start.Character, b.Start.Character))
}
