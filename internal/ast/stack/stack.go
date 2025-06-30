// Package stack provides AST functionality specific to terragrunt.stack.hcl files.
package stack

import (
	"terragrunt-ls/internal/ast"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// StackAST provides methods for working with terragrunt.stack.hcl files.
type StackAST interface {
	FindNodeAt(pos hcl.Pos) *ast.IndexedNode

	GetUnitLabel(node *ast.IndexedNode) (string, bool)
	GetStackLabel(node *ast.IndexedNode) (string, bool)
	GetUnitSource(node *ast.IndexedNode) (string, bool)
	GetUnitPath(node *ast.IndexedNode) (string, bool)
	GetStackSource(node *ast.IndexedNode) (string, bool)
	GetStackPath(node *ast.IndexedNode) (string, bool)
	FindUnitAt(pos hcl.Pos) (*ast.IndexedNode, bool)
	FindStackAt(pos hcl.Pos) (*ast.IndexedNode, bool)
}

// stackAST is the concrete implementation of StackAST
type stackAST struct {
	*ast.IndexedAST
}

// NewStackAST creates a new StackAST from an IndexedAST
func NewStackAST(indexedAST *ast.IndexedAST) StackAST {
	return &stackAST{IndexedAST: indexedAST}
}

// FindNodeAt returns the node at the given position in the file
func (s *stackAST) FindNodeAt(pos hcl.Pos) *ast.IndexedNode {
	return s.IndexedAST.FindNodeAt(pos)
}

// GetUnitLabel returns the label of the given node, if it is a unit block
func (s *stackAST) GetUnitLabel(node *ast.IndexedNode) (string, bool) {
	attr := ast.FindFirstParentMatch(node, ast.IsAttribute)
	if attr == nil {
		return "", false
	}

	unitBlock := ast.FindFirstParentMatch(attr, isUnitBlock)
	if unitBlock == nil {
		return "", false
	}

	name := ""
	if labels := unitBlock.Node.(*hclsyntax.Block).Labels; len(labels) > 0 {
		name = labels[0]
	}

	return name, true
}

// GetStackLabel returns the label of the given node, if it is a stack block
func (s *stackAST) GetStackLabel(node *ast.IndexedNode) (string, bool) {
	attr := ast.FindFirstParentMatch(node, ast.IsAttribute)
	if attr == nil {
		return "", false
	}

	stackBlock := ast.FindFirstParentMatch(attr, isStackBlock)
	if stackBlock == nil {
		return "", false
	}

	name := ""
	if labels := stackBlock.Node.(*hclsyntax.Block).Labels; len(labels) > 0 {
		name = labels[0]
	}

	return name, true
}

// GetUnitSource returns the source attribute value from a unit block
func (s *stackAST) GetUnitSource(node *ast.IndexedNode) (string, bool) {
	return s.getBlockAttribute(node, isUnitBlock, "source")
}

// GetUnitPath returns the path attribute value from a unit block
func (s *stackAST) GetUnitPath(node *ast.IndexedNode) (string, bool) {
	return s.getBlockAttribute(node, isUnitBlock, "path")
}

// GetStackSource returns the source attribute value from a stack block
func (s *stackAST) GetStackSource(node *ast.IndexedNode) (string, bool) {
	return s.getBlockAttribute(node, isStackBlock, "source")
}

// GetStackPath returns the path attribute value from a stack block
func (s *stackAST) GetStackPath(node *ast.IndexedNode) (string, bool) {
	return s.getBlockAttribute(node, isStackBlock, "path")
}

// FindUnitAt finds a unit block at the given position
func (s *stackAST) FindUnitAt(pos hcl.Pos) (*ast.IndexedNode, bool) {
	node := s.FindNodeAt(pos)
	if node == nil {
		return nil, false
	}

	unitBlock := ast.FindFirstParentMatch(node, isUnitBlock)

	return unitBlock, unitBlock != nil
}

// FindStackAt finds a stack block at the given position
func (s *stackAST) FindStackAt(pos hcl.Pos) (*ast.IndexedNode, bool) {
	node := s.FindNodeAt(pos)
	if node == nil {
		return nil, false
	}

	stackBlock := ast.FindFirstParentMatch(node, isStackBlock)

	return stackBlock, stackBlock != nil
}

// Helper functions

// isUnitBlock returns TRUE if the node is an HCL block of type "unit"
func isUnitBlock(inode *ast.IndexedNode) bool {
	block, ok := inode.Node.(*hclsyntax.Block)
	return ok && block.Type == "unit" && len(block.Labels) > 0
}

// isStackBlock returns TRUE if the node is an HCL block of type "stack"
func isStackBlock(inode *ast.IndexedNode) bool {
	block, ok := inode.Node.(*hclsyntax.Block)
	return ok && block.Type == "stack" && len(block.Labels) > 0
}

// getBlockAttribute is a helper to get attribute values from blocks
func (s *stackAST) getBlockAttribute(node *ast.IndexedNode, blockMatcher func(*ast.IndexedNode) bool, attrName string) (string, bool) {
	// First, try to find the attribute that contains the current node
	attr := ast.FindFirstParentMatch(node, ast.IsAttribute)
	if attr == nil {
		return "", false
	}

	// Check if the found attribute has the name we're looking for
	if attrNode, ok := attr.Node.(*hclsyntax.Attribute); ok {
		if attrNode.Name == attrName {
			// Verify we're within the correct block type
			block := ast.FindFirstParentMatch(attr, blockMatcher)
			if block != nil {
				// Extract the string value from the attribute expression
				return s.extractStringValue(attrNode.Expr)
			}
		}
	}

	return "", false
}

// extractStringValue extracts a string value from various HCL expression types
func (s *stackAST) extractStringValue(expr hclsyntax.Expression) (string, bool) {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		if e.Val.Type().FriendlyName() == "string" {
			return e.Val.AsString(), true
		}
	case *hclsyntax.TemplateExpr:
		// Handle quoted strings which are parsed as TemplateExpr
		if len(e.Parts) == 1 {
			if literal, ok := e.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
				if literal.Val.Type().FriendlyName() == "string" {
					return literal.Val.AsString(), true
				}
			}
		}
	}

	return "", false
}
