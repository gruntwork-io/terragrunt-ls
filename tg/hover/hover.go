// Package hover provides the logic for determining the target of a hover.
package hover

import (
	"log"
	"strings"
	"terragrunt-ls/tg/store"
	"terragrunt-ls/tg/text"

	"go.lsp.dev/protocol"
)

const (
	// HoverContextLocal is the context for a local hover.
	// This means that a hover is happening on top of a local variable.
	HoverContextLocal = "local"

	// HoverContextNull is the context for a null hover.
	// This means that a hover is happening on top of nothing useful.
	HoverContextNull = "null"
)

func GetHoverTargetWithContext(l *log.Logger, store store.Store, position protocol.Position) (string, string) {
	word := text.GetCursorWord(store.Document, position)
	if len(word) == 0 {
		l.Printf("No word found at %d:%d", position.Line, position.Character)

		return "", HoverContextNull
	}

	splitExpression := strings.Split(word, ".")

	const localPartsLen = 2

	if len(splitExpression) != localPartsLen {
		l.Printf("Invalid word found at %d:%d: %s", position.Line, position.Character, word)

		return "", HoverContextNull
	}

	if splitExpression[0] == "local" {
		l.Printf("Found local variable: %s", splitExpression[1])

		return splitExpression[1], HoverContextLocal
	}

	return "", HoverContextNull
}
