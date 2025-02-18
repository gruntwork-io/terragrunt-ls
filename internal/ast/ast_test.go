package ast_test

import (
	"terragrunt-ls/internal/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestIndexFileAST(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name            string
		contents        string
		expectedNodesAt map[protocol.Position]string
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
			expectedNodesAt: map[protocol.Position]string{
				{Line: 0, Character: 0}: "[1:1-3:2] *hclsyntax.Block",
			},
		},
		{
			name: "include",
			contents: `include "root" {
	path = "root.hcl"
}
`,
			expectedNodesAt: map[protocol.Position]string{
				{Line: 0, Character: 0}: "[1:1-3:2] *hclsyntax.Block",
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
			expectedNodesAt: map[protocol.Position]string{
				{Line: 0, Character: 0}: "[1:1-3:2] *hclsyntax.Block",
				{Line: 1, Character: 8}: "[2:9-2:24] *hclsyntax.ScopeTraversalExpr",
				{Line: 5, Character: 0}: "[5:8-7:2] *hclsyntax.Body",
				{Line: 8, Character: 0}: "[9:1-11:2] *hclsyntax.Attribute",
				{Line: 9, Character: 0}: "[9:10-11:2] *hclsyntax.ObjectConsExpr",
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.IndexFileAST("test.hcl", []byte(tt.contents))
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

func TestIsLocalAttribute(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		pos      protocol.Position
		expected bool
	}{
		{
			name: "not a local attribute",
			content: `inputs = {
	foo = "bar"
}`,
			pos:      protocol.Position{Line: 1, Character: 1},
			expected: false,
		},
		{
			name: "local attribute",
			content: `locals {
	foo = "bar"
}`,
			pos:      protocol.Position{Line: 1, Character: 1},
			expected: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.IndexFileAST("test.hcl", []byte(tt.content))
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
		pos      protocol.Position
		expected bool
	}{
		{
			name: "not a locals block",
			content: `inputs = {
	foo = "bar"
}`,
			pos:      protocol.Position{Line: 0, Character: 0},
			expected: false,
		},
		{
			name: "locals block",
			content: `locals {
	foo = "bar"
}`,
			pos:      protocol.Position{Line: 0, Character: 0},
			expected: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.IndexFileAST("test.hcl", []byte(tt.content))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			node := indexed.FindNodeAt(tt.pos)

			assert.Equal(t, tt.expected, ast.IsLocalsBlock(node.Node))
		})
	}
}

func TestIsIncludeBlock(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		pos      protocol.Position
		expected bool
	}{
		{
			name: "not an include block",
			content: `inputs = {
	foo = "bar"
}`,
			pos:      protocol.Position{Line: 0, Character: 0},
			expected: false,
		},
		{
			name: "include block",
			content: `include "root" {
	path = "root.hcl"
}`,
			pos:      protocol.Position{Line: 0, Character: 0},
			expected: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.IndexFileAST("test.hcl", []byte(tt.content))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			node := indexed.FindNodeAt(tt.pos)

			assert.Equal(t, tt.expected, ast.IsIncludeBlock(node.Node))
		})
	}
}

func TestIsAttribute(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		pos      protocol.Position
		expected bool
	}{
		{
			name: "not an attribute",
			content: `locals {
	foo = "bar"
}`,
			pos:      protocol.Position{Line: 0, Character: 0},
			expected: false,
		},
		{
			name: "attribute",
			content: `inputs = {
	foo = "bar"
}`,
			pos:      protocol.Position{Line: 0, Character: 0},
			expected: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.IndexFileAST("test.hcl", []byte(tt.content))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			node := indexed.FindNodeAt(tt.pos)

			assert.Equal(t, tt.expected, ast.IsAttribute(node.Node))
		})
	}
}

func TestGetNodeIncludePath(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		pos      protocol.Position
		expected string
	}{
		{
			name: "not an include block",
			content: `inputs = {
	foo = "bar"
}`,
			pos:      protocol.Position{Line: 0, Character: 0},
			expected: "",
		},
		{
			name: "include block beginning of path",
			content: `include "root" {
	path = "root.hcl"
}`,
			pos:      protocol.Position{Line: 1, Character: 1},
			expected: "root",
		},
		{
			name: "include block end of path",
			content: `include "root" {
	path = "root.hcl"
}`,
			pos:      protocol.Position{Line: 1, Character: 17},
			expected: "root",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.IndexFileAST("test.hcl", []byte(tt.content))
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

func TestGetLocalVariableName(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		pos      protocol.Position
		expected string
	}{
		{
			name: "not a local variable",
			content: `inputs = {
	foo = "bar"
}`,
			pos:      protocol.Position{Line: 0, Character: 0},
			expected: "",
		},
		{
			name: "local variable",
			content: `locals {
	foo = "bar"
}

inputs = {
	foo = local.foo
}`,

			pos:      protocol.Position{Line: 5, Character: 7},
			expected: "foo",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexed, err := ast.IndexFileAST("test.hcl", []byte(tt.content))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			node := indexed.FindNodeAt(tt.pos)

			name, ok := ast.GetLocalVariableName(node.Node)
			if tt.expected == "" {
				assert.False(t, ok)
				return
			}

			assert.True(t, ok)
			assert.Equal(t, tt.expected, name)
		})
	}
}
