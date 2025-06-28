// Package store provides the logic for the state stored for each document.
//
// Whenever possible, stored state should be used instead of re-parsing the document.
package store

import (
	"github.com/gruntwork-io/terragrunt/config"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"terragrunt-ls/internal/ast"
)

// Store represents the state for a standard terragrunt.hcl file
type Store struct {
	AST      *ast.IndexedAST
	Cfg      *config.TerragruntConfig
	CfgAsCty cty.Value
	Document string
}

// StackStore represents the state for a terragrunt.stack.hcl file
type StackStore struct {
	AST      *ast.IndexedAST
	StackCfg *config.StackConfig
	Document string
}

// ValuesStore represents the state for a terragrunt.values.hcl file
type ValuesStore struct {
	AST       *ast.IndexedAST
	ValuesHCL *hcl.File
	Document  string
}
