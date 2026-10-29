package definition_test

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg/definition"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

// FuzzGetDefinitionTargetWithContext tests that definition always returns a known context, and no name when there is nothing to jump to.
func FuzzGetDefinitionTargetWithContext(f *testing.F) {
	f.Add("include \"root\" {\n  path = find_in_parent_folders(\"root.hcl\")\n}\n", uint32(1), uint32(10))
	f.Add("dependency \"vpc\" {\n  config_path = \"../vpc\"\n}\n", uint32(1), uint32(20))
	f.Add("locals {\n  env  = \"dev\"\n  name = local.env\n}\n", uint32(2), uint32(15))

	f.Fuzz(func(t *testing.T, document string, line, character uint32) {
		indexed, err := ast.ParseHCLFile("/work/terragrunt.hcl", []byte(document))
		require.NotNil(t, indexed, "parse error: %v", err)

		st := store.Store{AST: indexed, Document: document, FileType: store.FileTypeUnit}
		position := protocol.Position{Line: line, Character: character}

		name, definitionContext := definition.GetDefinitionTargetWithContext(testutils.NewTestLogger(t), st, position)

		require.Contains(t, []string{
			definition.DefinitionContextLocal,
			definition.DefinitionContextInclude,
			definition.DefinitionContextDependency,
			definition.DefinitionContextNull,
		}, definitionContext)

		if definitionContext == definition.DefinitionContextNull {
			require.Empty(t, name)
		}
	})
}
