package ast

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// MinReferenceTraversalLen is the minimum number of steps a ScopeTraversalExpr
// must have to be a `<root>.<name>` reference (one root + one attribute).
const MinReferenceTraversalLen = 2

// ReferenceVisitor is invoked for each `<root>.<name>` reference found.
// r is the source range of the attribute step (just `<name>`, not the root).
type ReferenceVisitor func(expr *hclsyntax.ScopeTraversalExpr, r hcl.Range)

// WalkReferences walks body and invokes visitor for each ScopeTraversalExpr
// whose first traversal step is a TraverseRoot named root and whose second
// step is a TraverseAttr named name.
func WalkReferences(body *hclsyntax.Body, root, name string, visitor ReferenceVisitor) {
	if body == nil {
		return
	}

	_ = hclsyntax.VisitAll(body, func(node hclsyntax.Node) hcl.Diagnostics {
		expr, ok := node.(*hclsyntax.ScopeTraversalExpr)
		if !ok {
			return nil
		}

		if len(expr.Traversal) < MinReferenceTraversalLen {
			return nil
		}

		rootStep, ok := expr.Traversal[0].(hcl.TraverseRoot)
		if !ok || rootStep.Name != root {
			return nil
		}

		attrStep, ok := expr.Traversal[1].(hcl.TraverseAttr)
		if !ok || attrStep.Name != name {
			return nil
		}

		visitor(expr, TraverseAttrIdentRange(attrStep))

		return nil
	})
}

// TraverseAttrIdentRange returns the range of the attribute identifier alone,
// excluding the leading dot included in TraverseAttr.SrcRange.
func TraverseAttrIdentRange(step hcl.TraverseAttr) hcl.Range {
	r := step.SrcRange
	r.Start.Column++
	r.Start.Byte++

	return r
}
