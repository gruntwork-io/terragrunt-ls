// Package definition provides the logic for finding
// definitions in Terragrunt configurations.
package definition

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/tg/store"

	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

const (
	// DefinitionContextInclude is the context for an include definition.
	// This means that the user is trying to find the definition of an include.
	DefinitionContextInclude = "include"

	// DefinitionContextNull is the context for a null definition.
	// This means that the user is trying to go to the definition of nothing useful.
	DefinitionContextNull = "null"
)

func GetDefinitionTargetWithContext(l *zap.SugaredLogger, store store.Store, position protocol.Position) (string, string) {
	if store.AST == nil {
		l.Debugf("No AST found in store")

		return "", DefinitionContextNull
	}

	node := store.AST.FindNodeAt(position)

	if node == nil {
		l.Debugf("No node found at %d:%d", position.Line, position.Character)

		return "", DefinitionContextNull
	}

	label, ok := ast.GetNodeIncludeLabel(node)
	if !ok {
		l.Debugf("No include label found at %d:%d", position.Line, position.Character)

		return "", DefinitionContextNull
	}

	l.Debugf("Found include label: %s", label)

	return label, DefinitionContextInclude
}
