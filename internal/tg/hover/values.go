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
	// HoverContextValuesVariable is the context for hovering over a variable in a values file.
	HoverContextValuesVariable = "values_variable"

	// HoverContextValuesDependency is the context for hovering over a dependency reference.
	HoverContextValuesDependency = "values_dependency"
)

// GetValuesHoverTargetWithContext analyzes the position in a terragrunt.values.hcl file
// and returns hover information with a classifying context.
func GetValuesHoverTargetWithContext(l logger.Logger, s store.Store, position protocol.Position) (string, string) {
	if s.AST == nil {
		l.Debug("No AST found for values file")
		return "", HoverContextNull
	}

	word := text.GetCursorWord(s.Document, position)
	if len(word) == 0 {
		l.Debug("No word found at position", "line", position.Line, "character", position.Character)
		return "", HoverContextNull
	}

	if strings.Contains(word, ".") {
		parts := strings.Split(word, ".")
		if len(parts) >= 2 && parts[0] == "dependency" {
			depName := parts[1]
			l.Debug("Found dependency reference hover", "dependency", depName, "word", word)

			return depName, HoverContextValuesDependency
		}
	}

	l.Debug("Found potential variable hover", "variable", word)

	return word, HoverContextValuesVariable
}
