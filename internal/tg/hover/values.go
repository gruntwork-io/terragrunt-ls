// Package hover provides values-specific hover functionality.
package hover

import (
	"strings"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"

	"go.lsp.dev/protocol"
)

const (
	// HoverContextValuesVariable is the context for hovering over a variable in values block
	HoverContextValuesVariable = "values_variable"

	// HoverContextValuesDependency is the context for hovering over a dependency block
	HoverContextValuesDependency = "values_dependency"
)

// GetValuesHoverTargetWithContext analyzes the position in a values file and returns hover information
func GetValuesHoverTargetWithContext(l logger.Logger, store store.ValuesStore, position protocol.Position) (string, string) {
	if store.AST == nil {
		l.Debug("No AST found for values file")
		return "", HoverContextNull
	}

	// Get the word at the cursor position
	word := text.GetCursorWord(store.Document, position)
	if len(word) == 0 {
		l.Debug("No word found at position", "line", position.Line, "character", position.Character)
		return "", HoverContextNull
	}

	// Check if it's a dependency reference (dependency.name.output)
	if strings.Contains(word, ".") {
		parts := strings.Split(word, ".")
		if len(parts) >= 2 && parts[0] == "dependency" {
			depName := parts[1]
			l.Debug("Found dependency reference hover", "dependency", depName, "word", word)

			return depName, HoverContextValuesDependency
		}
	}

	// For now, treat any word as a potential variable
	// In the future, we could enhance this with AST analysis to be more precise
	l.Debug("Found potential variable hover", "variable", word)

	return word, HoverContextValuesVariable
}
