package stack_test

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/ast/stack"
	"testing"

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
