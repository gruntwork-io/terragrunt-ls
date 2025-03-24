// Package store provides the logic for the state stored for each document.
//
// Whenever possible, stored state should be used instead of re-parsing the document.
package store

import (
	"github.com/gruntwork-io/terragrunt/config"
	"github.com/zclconf/go-cty/cty"

	"terragrunt-ls/internal/ast"
)

type Store struct {
	Ast      *ast.IndexedAST
	Cfg      *config.TerragruntConfig
	CfgAsCty cty.Value
	Document string
}
