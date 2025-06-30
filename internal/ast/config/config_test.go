package config_test

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/ast/config"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigAST_Interface(t *testing.T) {
	t.Parallel()

	// Test HCL content with include and dependency blocks
	content := `
include "root" {
  path = find_in_parent_folders()
}

dependency "vpc" {
  config_path = "../vpc"
}

locals {
  env = "test"
}
`

	// Parse the content
	indexedAST, err := ast.ParseHCLFile("test.hcl", []byte(content))
	require.NoError(t, err)
	require.NotNil(t, indexedAST)

	// Create ConfigAST
	configAST := config.NewConfigAST(indexedAST)
	require.NotNil(t, configAST)

	// Test interface methods exist and work
	assert.NotNil(t, configAST.FindNodeAt)
	assert.NotNil(t, configAST.GetIncludeLabel)
	assert.NotNil(t, configAST.GetDependencyLabel)
	assert.NotNil(t, configAST.GetLocals)
	assert.NotNil(t, configAST.GetIncludes)

	// Test that locals and includes are captured
	locals := configAST.GetLocals()
	assert.NotNil(t, locals)

	includes := configAST.GetIncludes()
	assert.NotNil(t, includes)
}

func TestConfigAST_Methods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testFunc func(*testing.T, config.ConfigAST)
		name     string
		content  string
	}{
		{
			name: "basic functionality",
			content: `
include "root" {
  path = find_in_parent_folders()
}

locals {
  env = "test"
}
`,
			testFunc: func(t *testing.T, configAST config.ConfigAST) {
				t.Helper()
				// Basic test - just ensure it doesn't panic
				assert.NotNil(t, configAST)
			},
		},
		{
			name: "interface compliance",
			content: `
dependency "vpc" {
  config_path = "../vpc"
}
`,
			testFunc: func(t *testing.T, configAST config.ConfigAST) {
				t.Helper()
				// Test that it implements the interface
				var _ = configAST
				assert.NotNil(t, configAST)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indexedAST, err := ast.ParseHCLFile("test.hcl", []byte(tt.content))
			require.NoError(t, err)
			require.NotNil(t, indexedAST)

			configAST := config.NewConfigAST(indexedAST)
			require.NotNil(t, configAST)

			tt.testFunc(t, configAST)
		})
	}
}

func TestConfigAST_GetIncludeLabel(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		expected string
		pos      hcl.Pos
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

			configAST := config.NewConfigAST(indexed)
			node := configAST.FindNodeAt(tt.pos)

			label, ok := configAST.GetIncludeLabel(node)
			if tt.expected == "" {
				assert.False(t, ok)
				return
			}

			assert.True(t, ok)
			assert.Equal(t, tt.expected, label)
		})
	}
}

func TestConfigAST_GetDependencyLabel(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		content  string
		expected string
		pos      hcl.Pos
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

			configAST := config.NewConfigAST(indexed)
			node := configAST.FindNodeAt(tt.pos)

			label, ok := configAST.GetDependencyLabel(node)
			if tt.expected == "" {
				assert.False(t, ok)
				return
			}

			assert.True(t, ok)
			assert.Equal(t, tt.expected, label)
		})
	}
}

func TestNewConfigAST(t *testing.T) {
	t.Parallel()

	content := `
locals {
  region = "us-west-2"
  env    = "dev"
}

include "root" {
  path = find_in_parent_folders()
}

dependency "vpc" {
  config_path = "../vpc"
}
`
	indexed, err := ast.ParseHCLFile("test.hcl", []byte(content))
	require.NoError(t, err)

	configAST := config.NewConfigAST(indexed)
	require.NotNil(t, configAST)

	// Test interface methods
	assert.NotNil(t, configAST.GetLocals)
	assert.NotNil(t, configAST.GetIncludes)

	// Test that locals and includes are captured
	locals := configAST.GetLocals()
	assert.NotNil(t, locals)
	assert.Contains(t, locals, "region")
	assert.Contains(t, locals, "env")

	includes := configAST.GetIncludes()
	assert.NotNil(t, includes)
	assert.Contains(t, includes, "root", "Should contain 'root' include")
}

func TestConfigAST_GetIncludeLabel_NotIncludeAttribute(t *testing.T) {
	t.Parallel()

	content := `
locals {
  foo = "bar"
}
`
	indexed, err := ast.ParseHCLFile("test.hcl", []byte(content))
	require.NoError(t, err)

	configAST := config.NewConfigAST(indexed)

	// Find foo attribute (not in include block)
	fooNode := indexed.FindNodeAt(hcl.Pos{Line: 3, Column: 3})
	require.NotNil(t, fooNode)

	label, ok := configAST.GetIncludeLabel(fooNode)
	assert.False(t, ok)
	assert.Empty(t, label)
}

func TestConfigAST_GetDependencyLabel_NotConfigPath(t *testing.T) {
	t.Parallel()

	content := `
dependency "vpc" {
  other_attr = "value"
}
`
	indexed, err := ast.ParseHCLFile("test.hcl", []byte(content))
	require.NoError(t, err)

	configAST := config.NewConfigAST(indexed)

	// Find other_attr attribute (not config_path)
	attrNode := indexed.FindNodeAt(hcl.Pos{Line: 3, Column: 3})
	require.NotNil(t, attrNode)

	label, ok := configAST.GetDependencyLabel(attrNode)
	assert.False(t, ok)
	assert.Empty(t, label)
}
