package ast_test

import (
	"terragrunt-ls/internal/ast"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexFileAST(t *testing.T) {
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

			indexed, err := ast.IndexFileAST("test.hcl", []byte(tt.content))
			require.NoError(t, err)

			require.NotNil(t, indexed)

			node := indexed.FindNodeAt(tt.pos)

			path, ok := ast.GetNodeIncludePath(node)
			if tt.expected == "" {
				assert.False(t, ok)
				return
			}

			assert.True(t, ok)
			assert.Equal(t, tt.expected, path)
		})
	}
}
