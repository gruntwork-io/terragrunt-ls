// Package stack finds the `unit` and `stack` attributes of a terragrunt.stack.hcl file
// that go-to-definition can navigate from.
//
// Only top-level blocks match. A `unit` or `stack` block nested in another block, such as
// one a stack's `autoinclude` injects into the nested stack, is skipped: its `source` and
// `path` resolve against the nested stack, not the file being edited.
package stack

import (
	"terragrunt-ls/internal/ast"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// UnitSource returns the literal `source` of the top-level unit block whose `source`
// attribute contains node.
func UnitSource(node *ast.IndexedNode) (string, bool) {
	return literalSource(node, "unit")
}

// StackSource returns the literal `source` of the top-level stack block whose `source`
// attribute contains node.
func StackSource(node *ast.IndexedNode) (string, bool) {
	return literalSource(node, "stack")
}

// UnitPathLabel returns the label of the top-level unit block whose `path` attribute
// contains node. The `path` value may be any expression.
func UnitPathLabel(node *ast.IndexedNode) (string, bool) {
	return pathLabel(node, "unit")
}

// StackPathLabel returns the label of the top-level stack block whose `path` attribute
// contains node. The `path` value may be any expression.
func StackPathLabel(node *ast.IndexedNode) (string, bool) {
	return pathLabel(node, "stack")
}

// literalSource returns the string literal value of the `source` attribute that contains
// node, when that attribute belongs to a top-level block of blockType.
func literalSource(node *ast.IndexedNode, blockType string) (string, bool) {
	block, ok := attributeBlock(node, blockType, "source")
	if !ok {
		return "", false
	}

	return stringLiteral(block.Body.Attributes["source"].Expr)
}

// pathLabel returns the label of the top-level block of blockType whose `path` attribute
// contains node.
func pathLabel(node *ast.IndexedNode, blockType string) (string, bool) {
	block, ok := attributeBlock(node, blockType, "path")
	if !ok {
		return "", false
	}

	return block.Labels[0], true
}

// attributeBlock returns the labeled top-level block of blockType that directly holds the
// attribute named attrName containing node.
func attributeBlock(node *ast.IndexedNode, blockType, attrName string) (*hclsyntax.Block, bool) {
	attrNode := ast.FindFirstParentMatch(node, ast.IsAttribute)
	if attrNode == nil || attrNode.Node.(*hclsyntax.Attribute).Name != attrName {
		return nil, false
	}

	blockNode := ast.FindFirstParentMatch(attrNode, ast.IsBlock)
	if blockNode == nil {
		return nil, false
	}

	block := blockNode.Node.(*hclsyntax.Block)
	if block.Type != blockType || len(block.Labels) == 0 {
		return nil, false
	}

	if ast.FindFirstParentMatch(blockNode.Parent, ast.IsBlock) != nil {
		return nil, false
	}

	return block, true
}

// stringLiteral returns the value of expr when it is a plain string with no interpolation.
func stringLiteral(expr hclsyntax.Expression) (string, bool) {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		if e.Val.Type() == cty.String {
			return e.Val.AsString(), true
		}
	case *hclsyntax.TemplateExpr:
		if len(e.Parts) != 1 {
			return "", false
		}

		literal, ok := e.Parts[0].(*hclsyntax.LiteralValueExpr)
		if ok && literal.Val.Type() == cty.String {
			return literal.Val.AsString(), true
		}
	}

	return "", false
}
