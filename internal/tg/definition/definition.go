// Package definition provides the logic for finding
// definitions in Terragrunt configurations.
package definition

import (
	"terragrunt-ls/internal/ast"
	astconfig "terragrunt-ls/internal/ast/config"
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
	if store.AST == nil {
		l.Debug("No AST found")
		return "", DefinitionContextNull
	}

	cfgAST := astconfig.NewConfigAST(store.AST)

	node := cfgAST.FindNodeAt(ast.ToHCLPos(position))
	if node == nil {
		l.Debug("No node found at", "line", position.Line, "character", position.Character)
		return "", DefinitionContextNull
	}

	if include, ok := cfgAST.GetIncludeLabel(node); ok {
		l.Debug("Found include", "label", include)
		return include, DefinitionContextInclude
	}

	if dep, ok := cfgAST.GetDependencyLabel(node); ok {
		l.Debug("Found dependency", "label", dep)
		return dep, DefinitionContextDependency
	}

	l.Debug("No definition found at", "line", position.Line, "character", position.Character)

	return "", DefinitionContextNull
}
