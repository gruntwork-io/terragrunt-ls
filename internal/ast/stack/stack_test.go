package stack_test

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/ast/stack"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const stackFile = `
unit "database" {
  source = "./database"
  path   = "db"
}

unit "computed" {
  source = local.source
  path   = local.name
}

stack "nested" {
  source = "./nested-stack"
  path   = "nested"

  autoinclude {
    unit "injected" {
      source = "./injected"
      path   = "injected"
    }
  }
}
`

func TestStackAttributes(t *testing.T) {
	t.Parallel()

	indexedAST, err := ast.ParseHCLFile("terragrunt.stack.hcl", []byte(stackFile))
	require.NoError(t, err)

	tests := []struct {
		get      func(*ast.IndexedNode) (string, bool)
		name     string
		expected string
		line     int
		col      int
		found    bool
	}{
		{name: "unit source name", get: stack.UnitSource, line: 3, col: 3, expected: "./database", found: true},
		{name: "unit source value", get: stack.UnitSource, line: 3, col: 15, expected: "./database", found: true},
		{name: "unit source on path", get: stack.UnitSource, line: 4, col: 15},
		{name: "unit source on block header", get: stack.UnitSource, line: 2, col: 3},
		{name: "unit source non-literal", get: stack.UnitSource, line: 8, col: 15},
		{name: "unit source in stack block", get: stack.UnitSource, line: 13, col: 15},
		{name: "unit source in nested block", get: stack.UnitSource, line: 18, col: 17},
		{name: "unit path literal", get: stack.UnitPathLabel, line: 4, col: 15, expected: "database", found: true},
		{name: "unit path name", get: stack.UnitPathLabel, line: 4, col: 3, expected: "database", found: true},
		{name: "unit path computed", get: stack.UnitPathLabel, line: 9, col: 15, expected: "computed", found: true},
		{name: "unit path on source", get: stack.UnitPathLabel, line: 3, col: 15},
		{name: "unit path on block header", get: stack.UnitPathLabel, line: 2, col: 3},
		{name: "unit path in stack block", get: stack.UnitPathLabel, line: 14, col: 12},
		{name: "unit path in nested block", get: stack.UnitPathLabel, line: 19, col: 17},
		{name: "stack source", get: stack.StackSource, line: 13, col: 15, expected: "./nested-stack", found: true},
		{name: "stack source in unit block", get: stack.StackSource, line: 3, col: 15},
		{name: "stack source in nested block", get: stack.StackSource, line: 18, col: 17},
		{name: "stack path", get: stack.StackPathLabel, line: 14, col: 12, expected: "nested", found: true},
		{name: "stack path in unit block", get: stack.StackPathLabel, line: 4, col: 15},
		{name: "stack path in nested block", get: stack.StackPathLabel, line: 19, col: 17},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := indexedAST.FindNodeAt(hcl.Pos{Line: tt.line, Column: tt.col})
			require.NotNil(t, node)

			got, found := tt.get(node)
			assert.Equal(t, tt.found, found)
			assert.Equal(t, tt.expected, got)
		})
	}
}
