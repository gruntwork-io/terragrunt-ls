// Package hover provides the logic for determining the target of a hover.
package hover

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/tg/store"

	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

const (
	// HoverContextLocal is the context for a local hover.
	// This means that a hover is happening on top of a local variable.
	HoverContextLocal = "local"

	// HoverContextNull is the context for a null hover.
	// This means that a hover is happening on top of nothing useful.
	HoverContextNull = "null"
)

func GetHoverTargetWithContext(l *zap.SugaredLogger, store store.Store, position protocol.Position) (string, string) {
	if store.AST == nil {
		l.Debugf("No AST found in store")

		return "", HoverContextNull
	}

	node := store.AST.FindNodeAt(position)

	if node == nil {
		l.Debugf("No node found at %d:%d", position.Line, position.Character)

		return "", HoverContextNull
	}

	name, ok := ast.GetLocalVariableName(node.Node)
	if ok {
		l.Debugf("Found local variable: %s", name)

		return name, HoverContextLocal
	}

	return "", HoverContextNull
}
