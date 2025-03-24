// Package definition provides the logic for finding
// definitions in Terragrunt configurations.
package definition

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"

	"go.lsp.dev/protocol"
)

const (
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
	if store.Ast == nil {
		l.Debug("No AST found")
		return "", DefinitionContextNull
	}

	node := store.Ast.FindNodeAt(ast.ToHCLPos(position))
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

	l.Debug("No definition found at", "line", position.Line, "character", position.Character)

	return "", DefinitionContextNull
}
