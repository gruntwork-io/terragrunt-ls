package stack_test

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/ast/stack"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStackAST(t *testing.T, content string) *stack.StackAST {
	t.Helper()

	indexedAST, err := ast.ParseHCLFile("test.stack.hcl", []byte(content))
	require.NoError(t, err)

	return stack.NewStackAST(indexedAST)
}

func TestStackAST_UnitAttributes(t *testing.T) {
	t.Parallel()

	content := `
unit "database" {
  source = "./database"
  path   = "db"
}
`

	stackAST := newStackAST(t, content)

	tests := []struct {
		get      func(*stack.StackAST, *ast.IndexedNode) (string, bool)
		name     string
		expected string
		line     int
		col      int
		found    bool
	}{
		{
			name:     "source on attribute name",
			get:      (*stack.StackAST).GetUnitSource,
			line:     3,
			col:      3,
			expected: "./database",
			found:    true,
		},
		{
			name:     "source on attribute value",
			get:      (*stack.StackAST).GetUnitSource,
			line:     3,
			col:      15,
			expected: "./database",
			found:    true,
		},
		{name: "source getter on path attribute", get: (*stack.StackAST).GetUnitSource, line: 4, col: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := stackAST.FindNodeAt(hcl.Pos{Line: tt.line, Column: tt.col})
			require.NotNil(t, node)

			got, found := tt.get(stackAST, node)
			assert.Equal(t, tt.found, found)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestStackAST_StackAttributes(t *testing.T) {
	t.Parallel()

	content := `
unit "database" {
  source = "./database"
  path   = "db"
}

stack "nested" {
  source = "./nested-stack"
  path   = "nested"
}

stack "dynamic" {
  source = local.stack_source
  path   = "dynamic"
}
`

	stackAST := newStackAST(t, content)

	tests := []struct {
		get      func(*stack.StackAST, *ast.IndexedNode) (string, bool)
		name     string
		expected string
		line     int
		col      int
		found    bool
	}{
		{
			name:     "stack source",
			get:      (*stack.StackAST).GetStackSource,
			line:     8,
			col:      15,
			expected: "./nested-stack",
			found:    true,
		},
		{
			name:  "stack source getter on a path attribute",
			get:   (*stack.StackAST).GetStackSource,
			line:  9,
			col:   12,
			found: false,
		},
		{
			name:  "stack source getter inside a unit block",
			get:   (*stack.StackAST).GetStackSource,
			line:  3,
			col:   15,
			found: false,
		},
		{
			name:  "unit source getter inside a stack block",
			get:   (*stack.StackAST).GetUnitSource,
			line:  8,
			col:   15,
			found: false,
		},
		{
			name:  "non-literal source expression",
			get:   (*stack.StackAST).GetStackSource,
			line:  13,
			col:   15,
			found: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := stackAST.FindNodeAt(hcl.Pos{Line: tt.line, Column: tt.col})
			require.NotNil(t, node)

			got, found := tt.get(stackAST, node)
			assert.Equal(t, tt.found, found)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestStackAST_Labels(t *testing.T) {
	t.Parallel()

	content := `
unit "database" {
  source = "./database"
  path   = "db"
}

stack "nested" {
  source = "./nested-stack"
  path   = "nested"
}
`

	stackAST := newStackAST(t, content)

	tests := []struct {
		name          string
		expectedUnit  string
		expectedStack string
		line          int
		col           int
		unitFound     bool
		stackFound    bool
	}{
		{name: "unit block header", line: 2, col: 3, unitFound: true, expectedUnit: "database"},
		{name: "unit source value", line: 3, col: 15, unitFound: true, expectedUnit: "database"},
		{name: "stack block header", line: 7, col: 3, stackFound: true, expectedStack: "nested"},
		{name: "stack path value", line: 9, col: 12, stackFound: true, expectedStack: "nested"},
		{name: "outside any block", line: 1, col: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := stackAST.FindNodeAt(hcl.Pos{Line: tt.line, Column: tt.col})
			require.NotNil(t, node)

			unit, unitFound := stackAST.GetUnitLabel(node)
			assert.Equal(t, tt.unitFound, unitFound)
			assert.Equal(t, tt.expectedUnit, unit)

			stackLabel, stackFound := stackAST.GetStackLabel(node)
			assert.Equal(t, tt.stackFound, stackFound)
			assert.Equal(t, tt.expectedStack, stackLabel)
		})
	}
}

func TestStackAST_IsPathAttribute(t *testing.T) {
	t.Parallel()

	content := `
unit "database" {
  source = "./database"
  path   = "db"
}

unit "computed" {
  source = "./computed"
  path   = local.name
}

stack "nested" {
  source = "./nested-stack"
  path   = "nested"
}
`

	stackAST := newStackAST(t, content)

	isUnitPath := (*stack.StackAST).IsUnitPathAttribute
	isStackPath := (*stack.StackAST).IsStackPathAttribute

	tests := []struct {
		is       func(*stack.StackAST, *ast.IndexedNode) bool
		name     string
		line     int
		col      int
		expected bool
	}{
		{name: "unit literal path name", is: isUnitPath, line: 4, col: 3, expected: true},
		{name: "unit literal path value", is: isUnitPath, line: 4, col: 15, expected: true},
		{name: "unit computed path value", is: isUnitPath, line: 9, col: 18, expected: true},
		{name: "unit source attribute", is: isUnitPath, line: 3, col: 15},
		{name: "unit block header", is: isUnitPath, line: 2, col: 3},
		{name: "unit path checker inside a stack block", is: isUnitPath, line: 14, col: 12},
		{name: "stack literal path value", is: isStackPath, line: 14, col: 12, expected: true},
		{name: "stack source attribute", is: isStackPath, line: 13, col: 15},
		{name: "stack path checker inside a unit block", is: isStackPath, line: 4, col: 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := stackAST.FindNodeAt(hcl.Pos{Line: tt.line, Column: tt.col})
			require.NotNil(t, node)

			assert.Equal(t, tt.expected, tt.is(stackAST, node))
		})
	}
}
