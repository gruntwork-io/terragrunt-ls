// Package hover provides the logic for determining the target of a hover.
package hover

import (
	"strings"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"

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

func GetHoverTargetWithContext(l *logger.Logger, store store.Store, position protocol.Position) (string, string) {
	word := text.GetCursorWord(store.Document, position)
	if len(word) == 0 {
		l.Debug(
			"No word found",
			"line", position.Line,
			"character", position.Character,
		)

		return word, HoverContextNull
	}

	splitExpression := strings.Split(word, ".")

	const localPartsLen = 2

	if len(splitExpression) != localPartsLen {
		l.Debug(
			"Invalid word found",
			"line", position.Line,
			"character", position.Character,
			"word", word,
		)

		return word, HoverContextNull
	}

	if splitExpression[0] == "local" {
		l.Debug(
			"Found local variable",
			"line", position.Line,
			"character", position.Character,
			"local", splitExpression[1],
		)

		return splitExpression[1], HoverContextLocal
	}

	return word, HoverContextNull
}
