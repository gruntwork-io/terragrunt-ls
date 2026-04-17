// Package config provides AST functionality specific to standard terragrunt.hcl files.
package config

import (
	"terragrunt-ls/internal/ast"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ConfigAST provides methods for working with standard terragrunt.hcl files.
type ConfigAST interface {
	// Core AST methods
	FindNodeAt(pos hcl.Pos) *ast.IndexedNode

	// Config-specific methods
	GetIncludeLabel(node *ast.IndexedNode) (string, bool)
	GetDependencyLabel(node *ast.IndexedNode) (string, bool)

	// Access to scope information
	GetLocals() ast.Scope
	GetIncludes() ast.Scope
}

// configAST is the concrete implementation of ConfigAST
type configAST struct {
	*ast.IndexedAST
	includes ast.Scope
}

// NewConfigAST creates a new ConfigAST from an IndexedAST
func NewConfigAST(indexedAST *ast.IndexedAST) ConfigAST {
	c := &configAST{
		IndexedAST: indexedAST,
		includes:   make(ast.Scope),
	}

	// Build the includes scope by scanning the AST
	c.buildIncludesScope()

	return c
}

// buildIncludesScope scans the AST to build the includes scope
func (c *configAST) buildIncludesScope() {
	for _, nodes := range c.Index {
		for _, node := range nodes {
			if isIncludeBlock(node) {
				c.includes.Add(node)
			}
		}
	}
}

// isIncludeBlock returns TRUE if the node is an HCL block of type "include".
func isIncludeBlock(inode *ast.IndexedNode) bool {
	block, ok := inode.Node.(*hclsyntax.Block)
	return ok && block.Type == "include" && len(block.Labels) > 0
}

// isDependencyBlock returns TRUE if the node is an HCL block of type "dependency".
func isDependencyBlock(inode *ast.IndexedNode) bool {
	block, ok := inode.Node.(*hclsyntax.Block)
	return ok && block.Type == "dependency" && len(block.Labels) > 0
}

// FindNodeAt returns the node at the given position in the file
func (c *configAST) FindNodeAt(pos hcl.Pos) *ast.IndexedNode {
	return c.IndexedAST.FindNodeAt(pos)
}

// GetIncludeLabel returns the label of the given node, if it is an include block
func (c *configAST) GetIncludeLabel(node *ast.IndexedNode) (string, bool) {
	attr := ast.FindFirstParentMatch(node, ast.IsAttribute)
	if attr == nil {
		return "", false
	}

	includeBlock := ast.FindFirstParentMatch(attr, isIncludeBlock)
	if includeBlock == nil {
		return "", false
	}

	name := ""
	if labels := includeBlock.Node.(*hclsyntax.Block).Labels; len(labels) > 0 {
		name = labels[0]
	}

	return name, true
}

// GetDependencyLabel returns the label of the given node, if it is a dependency block
func (c *configAST) GetDependencyLabel(node *ast.IndexedNode) (string, bool) {
	attr := ast.FindFirstParentMatch(node, ast.IsAttribute)
	if attr == nil {
		return "", false
	}

	if attr.Node.(*hclsyntax.Attribute).Name != "config_path" {
		return "", false
	}

	depBlock := ast.FindFirstParentMatch(attr, isDependencyBlock)
	if depBlock == nil {
		return "", false
	}

	name := ""
	if labels := depBlock.Node.(*hclsyntax.Block).Labels; len(labels) > 0 {
		name = labels[0]
	}

	return name, true
}

// GetLocals returns the locals scope
func (c *configAST) GetLocals() ast.Scope {
	return c.Locals
}

// GetIncludes returns the includes scope
func (c *configAST) GetIncludes() ast.Scope {
	return c.includes
}
