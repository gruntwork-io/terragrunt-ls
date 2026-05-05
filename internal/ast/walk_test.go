package ast_test

import (
	"terragrunt-ls/internal/ast"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkReferences(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		contents string
		root     string
		ident    string
		expected []hcl.Range
	}{
		{
			name: "single local reference",
			contents: `locals {
  foo = "bar"
}

inputs = {
  v = local.foo
}
`,
			root:  "local",
			ident: "foo",
			expected: []hcl.Range{
				{Start: hcl.Pos{Line: 6, Column: 13}, End: hcl.Pos{Line: 6, Column: 16}},
			},
		},
		{
			name: "multiple references and unrelated traversals",
			contents: `inputs = {
  a = local.foo
  b = local.bar
  c = local.foo
  d = path.module
}
`,
			root:  "local",
			ident: "foo",
			expected: []hcl.Range{
				{Start: hcl.Pos{Line: 2, Column: 13}, End: hcl.Pos{Line: 2, Column: 16}},
				{Start: hcl.Pos{Line: 4, Column: 13}, End: hcl.Pos{Line: 4, Column: 16}},
			},
		},
		{
			name: "dependency outputs reference",
			contents: `inputs = {
  v = dependency.vpc.outputs.id
}
`,
			root:  "dependency",
			ident: "vpc",
			expected: []hcl.Range{
				{Start: hcl.Pos{Line: 2, Column: 18}, End: hcl.Pos{Line: 2, Column: 21}},
			},
		},
		{
			name: "no matches",
			contents: `inputs = {
  v = local.bar
}
`,
			root:     "local",
			ident:    "foo",
			expected: nil,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			iast, err := ast.ParseHCLFile("test.hcl", []byte(tt.contents))
			require.NoError(t, err)
			require.NotNil(t, iast.HCLFile)

			body, ok := iast.HCLFile.Body.(*hclsyntax.Body)
			require.True(t, ok)

			var got []hcl.Range
			ast.WalkReferences(body, tt.root, tt.ident, func(_ *hclsyntax.ScopeTraversalExpr, r hcl.Range) {
				got = append(got, r)
			})

			require.Len(t, got, len(tt.expected))
			for i := range tt.expected {
				assert.Equal(t, tt.expected[i].Start.Line, got[i].Start.Line, "start line %d", i)
				assert.Equal(t, tt.expected[i].Start.Column, got[i].Start.Column, "start col %d", i)
				assert.Equal(t, tt.expected[i].End.Line, got[i].End.Line, "end line %d", i)
				assert.Equal(t, tt.expected[i].End.Column, got[i].End.Column, "end col %d", i)
			}
		})
	}
}
