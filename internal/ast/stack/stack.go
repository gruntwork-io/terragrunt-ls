// Package stack provides AST functionality specific to terragrunt.stack.hcl files.
package stack

import (
	"terragrunt-ls/internal/ast"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// StackAST provides methods for working with terragrunt.stack.hcl files.
type StackAST struct {
	*ast.IndexedAST
}

// NewStackAST creates a new StackAST from an IndexedAST
func NewStackAST(indexedAST *ast.IndexedAST) *StackAST {
	return &StackAST{IndexedAST: indexedAST}
}

// GetUnitLabel returns the label of the given node, if it is a unit block
func (s *StackAST) GetUnitLabel(node *ast.IndexedNode) (string, bool) {
	return firstLabelFromContainingBlock(node, isUnitBlock)
}

// GetStackLabel returns the label of the given node, if it is a stack block
func (s *StackAST) GetStackLabel(node *ast.IndexedNode) (string, bool) {
	return firstLabelFromContainingBlock(node, isStackBlock)
}

// firstLabelFromContainingBlock walks up from the given node to the nearest
// block matching blockMatcher (including the node itself) and returns that
// block's first label.
func firstLabelFromContainingBlock(node *ast.IndexedNode, blockMatcher func(*ast.IndexedNode) bool) (string, bool) {
	block := ast.FindFirstParentMatch(node, blockMatcher)
	if block == nil {
		return "", false
	}

	return block.Node.(*hclsyntax.Block).Labels[0], true
}

// GetUnitSource returns the source attribute value from a unit block
func (s *StackAST) GetUnitSource(node *ast.IndexedNode) (string, bool) {
	return getBlockAttribute(node, isUnitBlock, "source")
}

// IsUnitPathAttribute reports whether the node is inside the path attribute of a unit block,
// regardless of whether the value is a literal.
func (s *StackAST) IsUnitPathAttribute(node *ast.IndexedNode) bool {
	return findBlockAttribute(node, isUnitBlock, "path") != nil
}

// GetStackSource returns the source attribute value from a stack block
func (s *StackAST) GetStackSource(node *ast.IndexedNode) (string, bool) {
	return getBlockAttribute(node, isStackBlock, "source")
}

// IsStackPathAttribute reports whether the node is inside the path attribute of a stack block,
// regardless of whether the value is a literal.
func (s *StackAST) IsStackPathAttribute(node *ast.IndexedNode) bool {
	return findBlockAttribute(node, isStackBlock, "path") != nil
}

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

// findBlockAttribute returns the attribute named attrName that contains the node,
// if that attribute belongs to a block matched by blockMatcher.
func findBlockAttribute(
	node *ast.IndexedNode,
	blockMatcher func(*ast.IndexedNode) bool,
	attrName string,
) *hclsyntax.Attribute {
	attr := ast.FindFirstParentMatch(node, ast.IsAttribute)
	if attr == nil {
		return nil
	}

	attrNode, ok := attr.Node.(*hclsyntax.Attribute)
	if !ok || attrNode.Name != attrName {
		return nil
	}

	if ast.FindFirstParentMatch(attr, blockMatcher) == nil {
		return nil
	}

	return attrNode
}

// getBlockAttribute returns the string literal value of the named attribute of a matched block.
func getBlockAttribute(
	node *ast.IndexedNode,
	blockMatcher func(*ast.IndexedNode) bool,
	attrName string,
) (string, bool) {
	attr := findBlockAttribute(node, blockMatcher, attrName)
	if attr == nil {
		return "", false
	}

	return extractStringValue(attr.Expr)
}

// extractStringValue extracts a string value from various HCL expression types
func extractStringValue(expr hclsyntax.Expression) (string, bool) {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		if e.Val.Type() == cty.String {
			return e.Val.AsString(), true
		}
	case *hclsyntax.TemplateExpr:
		if len(e.Parts) == 1 {
			if literal, ok := e.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
				if literal.Val.Type() == cty.String {
					return literal.Val.AsString(), true
				}
			}
		}
	}

	return "", false
}
