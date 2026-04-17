// Package store provides the logic for the state stored for each document.
//
// Whenever possible, stored state should be used instead of re-parsing the document.
package store

import (
	"github.com/gruntwork-io/terragrunt/pkg/config"
	"github.com/zclconf/go-cty/cty"

	"terragrunt-ls/internal/ast"
)

// FileType identifies the kind of Terragrunt configuration file.
type FileType int

const (
	// FileTypeUnknown is the file type for unrecognized files.
	FileTypeUnknown FileType = iota
	// FileTypeUnit is the default file type for terragrunt.hcl files.
	FileTypeUnit
	// FileTypeStack is the file type for terragrunt.stack.hcl files.
	FileTypeStack
	// FileTypeValues is the file type for terragrunt.values.hcl files.
	FileTypeValues
)

type Store struct {
	AST      *ast.IndexedAST
	Cfg      *config.TerragruntConfig
	StackCfg *config.StackConfig
	CfgAsCty cty.Value
	Document string
	FileType FileType
}
