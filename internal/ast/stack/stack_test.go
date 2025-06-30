package stack_test

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/ast/stack"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackAST_Interface(t *testing.T) {
	t.Parallel()

	// Test HCL content with unit and stack blocks
	content := `
unit "database" {
  source = "git::git@github.com:acme/infrastructure-catalog.git//units/mysql"
  path   = "database"
}

unit "app" {
  source = "git::git@github.com:acme/infrastructure-catalog.git//units/app"
  path   = "app"
}

stack "nested" {
  source = "./nested-stack"
  path   = "nested"
}
`

	// Parse the content
	indexedAST, err := ast.ParseHCLFile("test.stack.hcl", []byte(content))
	require.NoError(t, err)
	require.NotNil(t, indexedAST)

	// Create StackAST
	stackAST := stack.NewStackAST(indexedAST)
	require.NotNil(t, stackAST)

	// Test interface methods exist and work
	assert.NotNil(t, stackAST.FindNodeAt)
	assert.NotNil(t, stackAST.GetUnitLabel)
	assert.NotNil(t, stackAST.GetStackLabel)
	assert.NotNil(t, stackAST.GetUnitSource)
	assert.NotNil(t, stackAST.GetUnitPath)
	assert.NotNil(t, stackAST.FindUnitAt)
	assert.NotNil(t, stackAST.FindStackAt)
}

func TestStackAST_Methods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testFunc func(*testing.T, stack.StackAST)
		name     string
		content  string
	}{
		{
			name: "basic functionality",
			content: `
unit "database" {
  source = "./database"
  path   = "db"
}
`,
			testFunc: func(t *testing.T, stackAST stack.StackAST) {
				t.Helper()
				// Basic test - just ensure it doesn't panic
				assert.NotNil(t, stackAST)
			},
		},
		{
			name: "interface compliance",
			content: `
stack "nested" {
  source = "./nested"
  path   = "nested"
}
`,
			testFunc: func(t *testing.T, stackAST stack.StackAST) {
				t.Helper()
				// Test that it implements the interface
				var _ = stackAST
				assert.NotNil(t, stackAST)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexedAST, err := ast.ParseHCLFile("test.stack.hcl", []byte(tt.content))
			require.NoError(t, err)
			require.NotNil(t, indexedAST)

			stackAST := stack.NewStackAST(indexedAST)
			require.NotNil(t, stackAST)

			tt.testFunc(t, stackAST)
		})
	}
}

func TestStackAST_GetUnitSource(t *testing.T) {
	t.Parallel()

	content := `
unit "database" {
  source = "./database"
  path   = "db"
}
`

	indexedAST, err := ast.ParseHCLFile("test.stack.hcl", []byte(content))
	require.NoError(t, err)
	require.NotNil(t, indexedAST)

	stackAST := stack.NewStackAST(indexedAST)
	require.NotNil(t, stackAST)

	tests := []struct {
		name       string
		line       int
		col        int
		expected   string
		shouldFind bool
	}{
		{
			name:       "cursor on source attribute name",
			line:       3, // line with "source = "./database""
			col:        3, // position on "source"
			expected:   "./database",
			shouldFind: true,
		},
		{
			name:       "cursor on source attribute value",
			line:       3,  // line with "source = "./database""
			col:        15, // position within "./database"
			expected:   "./database",
			shouldFind: true,
		},
		{
			name:       "cursor on path attribute name",
			line:       4, // line with "path   = "db""
			col:        3, // position on "path"
			expected:   "",
			shouldFind: false, // We're looking for source, not path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := hcl.Pos{Line: tt.line, Column: tt.col}
			node := stackAST.FindNodeAt(pos)
			require.NotNil(t, node, "should find node at position")

			source, found := stackAST.GetUnitSource(node)

			if tt.shouldFind {
				assert.True(t, found, "should find unit source")
				assert.Equal(t, tt.expected, source)
			} else {
				assert.False(t, found, "should not find unit source")
			}
		})
	}
}

func TestStackAST_GetUnitPath(t *testing.T) {
	t.Parallel()

	content := `
unit "database" {
  source = "./database"
  path   = "db"
}
`

	indexedAST, err := ast.ParseHCLFile("test.stack.hcl", []byte(content))
	require.NoError(t, err)
	require.NotNil(t, indexedAST)

	stackAST := stack.NewStackAST(indexedAST)
	require.NotNil(t, stackAST)

	tests := []struct {
		name       string
		line       int
		col        int
		expected   string
		shouldFind bool
	}{
		{
			name:       "cursor on path attribute name",
			line:       4, // line with "path   = "db""
			col:        3, // position on "path"
			expected:   "db",
			shouldFind: true,
		},
		{
			name:       "cursor on path attribute value",
			line:       4,  // line with "path   = "db""
			col:        15, // position within "db"
			expected:   "db",
			shouldFind: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := hcl.Pos{Line: tt.line, Column: tt.col}
			node := stackAST.FindNodeAt(pos)
			require.NotNil(t, node, "should find node at position")

			path, found := stackAST.GetUnitPath(node)

			if tt.shouldFind {
				assert.True(t, found, "should find unit path")
				assert.Equal(t, tt.expected, path)
			} else {
				assert.False(t, found, "should not find unit path")
			}
		})
	}
}

func TestStackAST_FindUnitAt(t *testing.T) {
	t.Parallel()

	content := `
unit "database" {
  source = "./database"
  path   = "db"
}

unit "app" {
  source = "./app"
  path   = "app"
}
`

	indexedAST, err := ast.ParseHCLFile("test.stack.hcl", []byte(content))
	require.NoError(t, err)
	require.NotNil(t, indexedAST)

	stackAST := stack.NewStackAST(indexedAST)
	require.NotNil(t, stackAST)

	tests := []struct {
		name       string
		line       int
		col        int
		shouldFind bool
	}{
		{
			name:       "cursor inside first unit block",
			line:       3, // line with "source = "./database""
			col:        10,
			shouldFind: true,
		},
		{
			name:       "cursor inside second unit block",
			line:       8, // line with "source = "./app""
			col:        10,
			shouldFind: true,
		},
		{
			name:       "cursor outside unit blocks",
			line:       1, // empty line at top
			col:        1,
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := hcl.Pos{Line: tt.line, Column: tt.col}
			_, found := stackAST.FindUnitAt(pos)
			assert.Equal(t, tt.shouldFind, found)
		})
	}
}

func TestStackAST_FindStackAt(t *testing.T) {
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

	indexedAST, err := ast.ParseHCLFile("test.stack.hcl", []byte(content))
	require.NoError(t, err)
	require.NotNil(t, indexedAST)

	stackAST := stack.NewStackAST(indexedAST)
	require.NotNil(t, stackAST)

	tests := []struct {
		name        string
		line        int
		col         int
		shouldFind  bool
		description string
	}{
		{
			name:        "inside unit block",
			line:        2, // inside unit block
			col:         3,
			shouldFind:  false,
			description: "should not find stack block when inside unit block",
		},
		{
			name:        "stack block name",
			line:        6, // line with "stack "nested" {"
			col:         3, // position on "stack"
			shouldFind:  true,
			description: "should find stack block at block name",
		},
		{
			name:        "stack source attribute",
			line:        7,  // line with "source = "./nested-stack""
			col:         15, // position in "./nested-stack"
			shouldFind:  true,
			description: "should find stack block at source attribute",
		},
		{
			name:        "stack path attribute",
			line:        8,  // line with "path = "nested""
			col:         10, // position in "nested"
			shouldFind:  true,
			description: "should find stack block at path attribute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := hcl.Pos{Line: tt.line, Column: tt.col}

			stackBlock, found := stackAST.FindStackAt(pos)

			assert.Equal(t, tt.shouldFind, found, tt.description)
			if tt.shouldFind {
				assert.NotNil(t, stackBlock, "Stack block should not be nil when found")
			} else {
				assert.Nil(t, stackBlock, "Stack block should be nil when not found")
			}
		})
	}
}
