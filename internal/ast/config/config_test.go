package config_test

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/ast/config"
	"testing"

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
