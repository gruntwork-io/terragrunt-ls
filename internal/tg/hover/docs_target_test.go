package hover_test

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/docs"
	"terragrunt-ls/internal/tg/hover"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestGetDocsTarget(t *testing.T) {
	t.Parallel()

	const document = `terraform {
  source = "../modules/vpc"
}

inputs = {
  tag = get_env("TAG", "latest")
}

locals {
  inputs = "not the attribute"
}
`

	tc := []struct {
		name     string
		kind     docs.Kind
		target   string
		position protocol.Position
		found    bool
	}{
		{
			name:     "block type",
			position: protocol.Position{Line: 0, Character: 3},
			kind:     docs.KindBlock,
			target:   "terraform",
			found:    true,
		},
		{name: "block brace", position: protocol.Position{Line: 0, Character: 10}},
		{name: "attribute in a block", position: protocol.Position{Line: 1, Character: 4}},
		{
			name:     "top-level attribute",
			position: protocol.Position{Line: 4, Character: 0},
			kind:     docs.KindAttribute,
			target:   "inputs",
			found:    true,
		},
		{
			name:     "function name",
			position: protocol.Position{Line: 5, Character: 10},
			kind:     docs.KindFunction,
			target:   "get_env",
			found:    true,
		},
		{name: "function argument", position: protocol.Position{Line: 5, Character: 18}},
		{name: "local named like an attribute", position: protocol.Position{Line: 9, Character: 3}},
	}

	indexed, err := ast.ParseHCLFile("terragrunt.hcl", []byte(document))
	require.NoError(t, err)

	st := &store.Store{AST: indexed, Document: document, FileType: store.FileTypeUnit}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			kind, target, found := hover.GetDocsTarget(st, tt.position)
			assert.Equal(t, tt.found, found)
			assert.Equal(t, tt.kind, kind)
			assert.Equal(t, tt.target, target)
		})
	}
}

func TestGetDocsTargetWithoutAST(t *testing.T) {
	t.Parallel()

	_, _, found := hover.GetDocsTarget(&store.Store{}, protocol.Position{})
	assert.False(t, found)
}
