// Package hover provides the logic for determining the target of a hover.
package hover

import (
	"strings"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"

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
	word := text.GetCursorWord(store.Document, position)
	if len(word) == 0 {
		l.Debugf("No word found at %d:%d", position.Line, position.Character)

		return word, HoverContextNull
	}

	splitExpression := strings.Split(word, ".")

	const localPartsLen = 2

	if len(splitExpression) != localPartsLen {
		l.Debugf("Invalid word found at %d:%d: %s", position.Line, position.Character, word)

		return word, HoverContextNull
	}

	if splitExpression[0] == "local" {
		l.Debugf("Found local variable: %s", splitExpression[1])

		return splitExpression[1], HoverContextLocal
	}

	return word, HoverContextNull
}
