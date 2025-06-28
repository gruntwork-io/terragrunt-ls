// Package config provides AST functionality specific to standard terragrunt.hcl files.
package config

import (
	"terragrunt-ls/internal/ast"

	"github.com/hashicorp/hcl/v2"
)

// ConfigAST provides methods for working with standard terragrunt.hcl files.
type ConfigAST interface {
	FindNodeAt(pos hcl.Pos) *ast.IndexedNode

	GetIncludeLabel(node *ast.IndexedNode) (string, bool)
	GetDependencyLabel(node *ast.IndexedNode) (string, bool)

	GetLocals() ast.Scope
	GetIncludes() ast.Scope
}

// configAST is the concrete implementation of ConfigAST
type configAST struct {
	*ast.IndexedAST
}

// NewConfigAST creates a new ConfigAST from an IndexedAST
func NewConfigAST(indexedAST *ast.IndexedAST) ConfigAST {
	return &configAST{IndexedAST: indexedAST}
}

// FindNodeAt returns the node at the given position in the file
func (c *configAST) FindNodeAt(pos hcl.Pos) *ast.IndexedNode {
	return c.IndexedAST.FindNodeAt(pos)
}

// GetIncludeLabel returns the label of the given node, if it is an include block
func (c *configAST) GetIncludeLabel(node *ast.IndexedNode) (string, bool) {
	return ast.GetNodeIncludeLabel(node)
}

// GetDependencyLabel returns the label of the given node, if it is a dependency block
func (c *configAST) GetDependencyLabel(node *ast.IndexedNode) (string, bool) {
	return ast.GetNodeDependencyLabel(node)
}

// GetLocals returns the locals scope
func (c *configAST) GetLocals() ast.Scope {
	return c.IndexedAST.Locals
}

// GetIncludes returns the includes scope
func (c *configAST) GetIncludes() ast.Scope {
	return c.IndexedAST.Includes
}
