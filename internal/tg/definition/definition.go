// Package definition provides the logic for finding
// definitions in Terragrunt configurations.
package definition

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"go.lsp.dev/protocol"
)

const (
	// DefinitionContextLocal is the context for a local variable definition.
	// This means that the user is trying to find the definition of a `local.X`
	// reference, which resolves to a `locals { X = ... }` declaration in the
	// current file or a sibling file in the same module folder.
	DefinitionContextLocal = "local"

	// DefinitionContextInclude is the context for an include definition.
	// This means that the user is trying to find the definition of an include.
	DefinitionContextInclude = "include"

	// DefinitionContextDependency is the context for a dependency definition.
	// This means that the user is trying to find the definition of a dependency.
	DefinitionContextDependency = "dependency"

	// DefinitionContextNull is the context for a null definition.
	// This means that the user is trying to go to the definition of nothing useful.
	DefinitionContextNull = "null"
)

func GetDefinitionTargetWithContext(l logger.Logger, store store.Store, position protocol.Position) (string, string) {
	if store.AST == nil {
		l.Debug("No AST found")
		return "", DefinitionContextNull
	}

	node := store.AST.FindNodeAt(ast.ToHCLPos(position))
	if node == nil {
		l.Debug("No node found at", "line", position.Line, "character", position.Character)
		return "", DefinitionContextNull
	}

	if include, ok := ast.GetNodeIncludeLabel(node); ok {
		l.Debug("Found include", "label", include)
		return include, DefinitionContextInclude
	}

	if dep, ok := ast.GetNodeDependencyLabel(node); ok {
		l.Debug("Found dependency", "label", dep)
		return dep, DefinitionContextDependency
	}

	if expr, ok := node.Node.(*hclsyntax.ScopeTraversalExpr); ok {
		if name, context, ok := traversalDefinitionTarget(expr); ok {
			l.Debug("Found traversal target", "name", name, "context", context)
			return name, context
		}
	}

	l.Debug("No definition found at", "line", position.Line, "character", position.Character)

	return "", DefinitionContextNull
}

// traversalDefinitionTarget extracts a (name, context) pair from a
// `<root>.<name>` traversal where root is one of local/include/dependency.
func traversalDefinitionTarget(expr *hclsyntax.ScopeTraversalExpr) (string, string, bool) {
	if len(expr.Traversal) < ast.MinReferenceTraversalLen {
		return "", "", false
	}

	rootStep, ok := expr.Traversal[0].(hcl.TraverseRoot)
	if !ok {
		return "", "", false
	}

	attrStep, ok := expr.Traversal[1].(hcl.TraverseAttr)
	if !ok {
		return "", "", false
	}

	switch rootStep.Name {
	case "local":
		return attrStep.Name, DefinitionContextLocal, true
	case "include":
		return attrStep.Name, DefinitionContextInclude, true
	case "dependency":
		return attrStep.Name, DefinitionContextDependency, true
	}

	return "", "", false
}
