package ast_test

import (
	"terragrunt-ls/internal/ast"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHCLFile(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name            string
		contents        string
		expectedNodesAt map[hcl.Pos]string
	}{
		{
			name:     "empty hcl",
			contents: ``,
		},
		{
			name: "locals",
			contents: `locals {
	foo = "bar"
}
`,
			expectedNodesAt: map[hcl.Pos]string{
				{Line: 1, Column: 1}: "[1:1-3:2] *hclsyntax.Block",
			},
		},
		{
			name: "include",
			contents: `include "root" {
	path = "root.hcl"
}
`,
			expectedNodesAt: map[hcl.Pos]string{
				{Line: 1, Column: 1}: "[1:1-3:2] *hclsyntax.Block",
			},
		},
		{
			name: "include with locals and inputs",
			contents: `include "root" {
	path = local.root_path
}
		
locals {
	root_path = "root.hcl"
}

inputs = {
	foo = "bar"
}
`,
			expectedNodesAt: map[hcl.Pos]string{
				{Line: 1, Column: 1}:  "[1:1-3:2] *hclsyntax.Block",
				{Line: 6, Column: 1}:  "[5:8-7:2] *hclsyntax.Body",
				{Line: 9, Column: 1}:  "[9:1-11:2] *hclsyntax.Attribute",
				{Line: 10, Column: 1}: "[9:10-11:2] *hclsyntax.ObjectConsExpr",
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.ParseHCLFile("test.hcl", []byte(tt.contents))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			if tt.expectedNodesAt == nil {
				return
			}

			for pos, expected := range tt.expectedNodesAt {
				node := indexed.FindNodeAt(pos)
				require.NotNil(t, node)

				assert.Equal(t, expected, node.String())
			}
		})
	}
}

func TestParseHCLFile_WithErrors(t *testing.T) {
	t.Parallel()

	content := `locals {
		foo = "bar
	}`

	indexed, err := ast.ParseHCLFile("test.hcl", []byte(content))

	// We should still get a partially indexed AST
	assert.NotNil(t, indexed)
	// And the error should be from the HCL parser
	assert.Error(t, err)
}

func TestIndexedAST_FindNodeAt_BasicCases(t *testing.T) {
	t.Parallel()

	// Test with empty file
	t.Run("empty file", func(t *testing.T) {
		t.Parallel()
		indexed, err := ast.ParseHCLFile("test.hcl", []byte(``))
		require.NoError(t, err)
		require.NotNil(t, indexed)
		node := indexed.FindNodeAt(hcl.Pos{Line: 1, Column: 1})
		assert.Nil(t, node, "Should not find a node in an empty file")
	})

	// Test with position within a node span
	t.Run("position within node span", func(t *testing.T) {
		t.Parallel()
		content := `locals {
		foo = "bar"
	}`
		indexed, err := ast.ParseHCLFile("test.hcl", []byte(content))
		require.NoError(t, err)
		require.NotNil(t, indexed)
		node := indexed.FindNodeAt(hcl.Pos{Line: 2, Column: 1})
		assert.NotNil(t, node, "Should find a node within the node span")
	})
}

func TestIsLocalAttribute(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		pos      hcl.Pos
		expected bool
	}{
		{
			name: "not a local attribute",
			content: `inputs = {
	foo = "bar"
}`,
			pos:      hcl.Pos{Line: 2, Column: 2},
			expected: false,
		},
		{
			name: "local attribute",
			content: `locals {
	foo = "bar"
}`,
			pos:      hcl.Pos{Line: 2, Column: 2},
			expected: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.ParseHCLFile("test.hcl", []byte(tt.content))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			node := indexed.FindNodeAt(tt.pos)

			assert.Equal(t, tt.expected, ast.IsLocalAttribute(node))
		})
	}
}

func TestIsLocalsBlock(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		pos      hcl.Pos
		expected bool
	}{
		{
			name: "not a locals block",
			content: `inputs = {
	foo = "bar"
}`,
			pos:      hcl.Pos{Line: 1, Column: 1},
			expected: false,
		},
		{
			name: "locals block",
			content: `locals {
	foo = "bar"
}`,
			pos:      hcl.Pos{Line: 1, Column: 1},
			expected: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.ParseHCLFile("test.hcl", []byte(tt.content))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			node := indexed.FindNodeAt(tt.pos)

			assert.Equal(t, tt.expected, ast.IsLocalsBlock(node))
		})
	}
}

func TestIsIncludeBlock(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		pos      hcl.Pos
		expected bool
	}{
		{
			name: "not an include block",
			content: `inputs = {
	foo = "bar"
}`,
			pos:      hcl.Pos{Line: 1, Column: 1},
			expected: false,
		},
		{
			name: "include block",
			content: `include "root" {
	path = "root.hcl"
}`,
			pos:      hcl.Pos{Line: 1, Column: 1},
			expected: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.ParseHCLFile("test.hcl", []byte(tt.content))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			node := indexed.FindNodeAt(tt.pos)

			assert.Equal(t, tt.expected, ast.IsIncludeBlock(node))
		})
	}
}

func TestIsAttribute(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		pos      hcl.Pos
		expected bool
	}{
		{
			name: "not an attribute",
			content: `locals {
	foo = "bar"
}`,
			pos:      hcl.Pos{Line: 1, Column: 1},
			expected: false,
		},
		{
			name: "attribute",
			content: `inputs = {
	foo = "bar"
}`,
			pos:      hcl.Pos{Line: 1, Column: 1},
			expected: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.ParseHCLFile("test.hcl", []byte(tt.content))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			node := indexed.FindNodeAt(tt.pos)

			assert.Equal(t, tt.expected, ast.IsAttribute(node))
		})
	}
}

func TestGetNodeIncludeLabel(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		pos      hcl.Pos
		expected string
	}{
		{
			name: "not an include block",
			content: `inputs = {
	foo = "bar"
}`,
			pos:      hcl.Pos{Line: 1, Column: 1},
			expected: "",
		},
		{
			name: "include block beginning of path",
			content: `include "root" {
	path = "root.hcl"
}`,
			pos:      hcl.Pos{Line: 2, Column: 2},
			expected: "root",
		},
		{
			name: "include block end of path",
			content: `include "root" {
	path = "root.hcl"
}`,
			pos:      hcl.Pos{Line: 2, Column: 18},
			expected: "root",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.ParseHCLFile("test.hcl", []byte(tt.content))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			node := indexed.FindNodeAt(tt.pos)

			path, ok := ast.GetNodeIncludeLabel(node)
			if tt.expected == "" {
				assert.False(t, ok)
				return
			}

			assert.True(t, ok)
			assert.Equal(t, tt.expected, path)
		})
	}
}

func TestGetNodeDependencyLabel(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		pos      hcl.Pos
		expected string
	}{
		{
			name: "not a dependency block",
			content: `inputs = {
	foo = "bar"
}`,
			pos:      hcl.Pos{Line: 1, Column: 1},
			expected: "",
		},
		{
			name: "dependency block beginning of path",
			content: `dependency "vpc" {
	config_path = "../vpc"
}`,
			pos:      hcl.Pos{Line: 2, Column: 2},
			expected: "vpc",
		},
		{
			name: "dependency block end of path",
			content: `dependency "vpc" {
	config_path = "../vpc"
}`,
			pos:      hcl.Pos{Line: 2, Column: 18},
			expected: "vpc",
		},
		{
			name: "dependency block wrong attribute",
			content: `dependency "vpc" {
	other_field = "../vpc"
}`,
			pos:      hcl.Pos{Line: 2, Column: 18},
			expected: "",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.ParseHCLFile("test.hcl", []byte(tt.content))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			node := indexed.FindNodeAt(tt.pos)

			path, ok := ast.GetNodeDependencyLabel(node)
			if tt.expected == "" {
				assert.False(t, ok)
				return
			}

			assert.True(t, ok)
			assert.Equal(t, tt.expected, path)
		})
	}
}

func TestFindFirstParentMatch(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		pos      hcl.Pos
		matcher  func(node *ast.IndexedNode) bool
		expected bool
	}{
		{
			name: "find attribute parent",
			content: `locals {
		foo = "bar"
	}`,
			pos:      hcl.Pos{Line: 2, Column: 2},
			matcher:  ast.IsLocalsBlock,
			expected: true,
		},
		{
			name: "no matching parent",
			content: `locals {
		foo = "bar"
	}`,
			pos:      hcl.Pos{Line: 2, Column: 2},
			matcher:  ast.IsDependencyBlock,
			expected: false,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.ParseHCLFile("test.hcl", []byte(tt.content))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			node := indexed.FindNodeAt(tt.pos)
			require.NotNil(t, node)

			match := ast.FindFirstParentMatch(node, tt.matcher)
			if !tt.expected {
				assert.Nil(t, match)
			} else {
				assert.NotNil(t, match)
			}
		})
	}
}
func TestScope_Add(t *testing.T) {
	t.Parallel()

	// Test manual creation of scope with a block node
	t.Run("manually created scope with block", func(t *testing.T) {
		t.Parallel()

		// Create a scope
		scope := ast.Scope{}

		// Parse some HCL with a block
		content := `include "root" {
		path = "root.hcl"
	}`

		indexed, err := ast.ParseHCLFile("test.hcl", []byte(content))
		require.NoError(t, err)
		require.NotNil(t, indexed)

		// Find the block node
		node := indexed.FindNodeAt(hcl.Pos{Line: 1, Column: 1})
		require.NotNil(t, node)

		// Add to the scope directly
		if block, ok := node.Node.(*hclsyntax.Block); ok && len(block.Labels) > 0 {
			scope[block.Labels[0]] = node

			// Verify it was added with the correct key
			assert.Len(t, scope, 1)
			assert.Contains(t, scope, "root")
		}
	})
}

// Test that include and local scopes are updated in parsing
func TestIndexedAST_Scopes(t *testing.T) {
	t.Parallel()

	content := `
locals {
  region = "us-west-2"
  env    = "dev"
}

include "root" {
  path = find_in_parent_folders()
}
`
	indexed, err := ast.ParseHCLFile("test.hcl", []byte(content))
	require.NoError(t, err)

	// Test locals scope
	locals := indexed.Locals
	assert.NotNil(t, locals, "Locals scope should not be nil")
	assert.Contains(t, locals, "region", "Should contain 'region' local")
	assert.Contains(t, locals, "env", "Should contain 'env' local")

	// Test includes scope existence
	includes := indexed.Includes
	assert.NotNil(t, includes, "Includes scope should not be nil")
	assert.Contains(t, includes, "root", "Should contain 'root' include")
}
