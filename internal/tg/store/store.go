// Package store provides the logic for the state stored for each document.
//
// Whenever possible, stored state should be used instead of re-parsing the document.
package store

import (
	"github.com/gruntwork-io/terragrunt/pkg/config"
	"github.com/zclconf/go-cty/cty"

	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/docs"
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

// DocsConfig returns the kind of configuration whose reference docs apply to
// files of type t. A file of unknown type is taken to be one a unit includes.
func (t FileType) DocsConfig() (docs.Config, bool) {
	switch t {
	case FileTypeUnit, FileTypeUnknown:
		return docs.ConfigUnit, true
	case FileTypeStack:
		return docs.ConfigStack, true
	case FileTypeValues:
		return "", false
	}

	return "", false
}

type Store struct {
	AST      *ast.IndexedAST
	Cfg      *config.TerragruntConfig
	StackCfg *config.StackConfig
	CfgAsCty cty.Value
	Document string
	FileType FileType
}
