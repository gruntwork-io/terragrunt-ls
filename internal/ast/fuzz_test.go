package ast_test

import (
	"terragrunt-ls/internal/ast"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/require"
)

// FuzzParseHCLFile tests that the index is always built and that a node lookup never returns a node from a later line.
func FuzzParseHCLFile(f *testing.F) {
	f.Add([]byte("locals {\n  env  = \"dev\"\n  name = \"${local.env}-app\"\n}\n"), 3, 15)
	f.Add([]byte("include \"root\" {\n  path = find_in_parent_folders(\"root.hcl\")\n}\n"), 2, 10)
	f.Add([]byte("dependency \"vpc\" {\n  config_path = \"../vpc\"\n}\n"), 2, 5)
	f.Add([]byte("include {\n  path = \"x\"\n}\n"), 2, 3)
	f.Add([]byte("locals {"), 1, 1)

	f.Fuzz(func(t *testing.T, contents []byte, line, column int) {
		if line < 1 || column < 1 {
			t.Skip("HCL positions start at line 1, column 1")
		}

		indexed, err := ast.ParseHCLFile("terragrunt.hcl", contents)
		require.NotNil(t, indexed, "parse error: %v", err)

		node := indexed.FindNodeAt(hcl.Pos{Line: line, Column: column})
		if node == nil {
			return
		}

		// An Attributes node reports the range of an arbitrary member, so its start line says nothing about the lookup.
		if _, ok := node.Node.(hclsyntax.Attributes); ok {
			return
		}

		require.LessOrEqual(t, node.Range().Start.Line, line)

		if ast.IsLocalAttribute(node) {
			require.IsType(t, &hclsyntax.Attribute{}, node.Node)
		}
	})
}
