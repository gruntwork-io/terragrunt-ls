// Package store provides the logic for the state stored for each document.
//
// Whenever possible, stored state should be used instead of re-parsing the document.
package store

import (
	"github.com/gruntwork-io/terragrunt/config"
	"github.com/gruntwork-io/terragrunt/config/hclparse"
	"github.com/zclconf/go-cty/cty"

	"terragrunt-ls/internal/ast"
	astconfig "terragrunt-ls/internal/ast/config"
	aststack "terragrunt-ls/internal/ast/stack"
)

// Store represents the state for a standard terragrunt.hcl file
type Store struct {
	AST      astconfig.ConfigAST
	Cfg      *config.TerragruntConfig
	CfgAsCty cty.Value
	Document string
}

// StackStore represents the state for a terragrunt.stack.hcl file
type StackStore struct {
	AST      aststack.StackAST
	StackCfg *config.StackConfig
	Document string
}

// ValuesStore represents the state for a terragrunt.values.hcl file
type ValuesStore struct {
	AST       *ast.IndexedAST
	ValuesHCL *hclparse.File
	Document  string
}
