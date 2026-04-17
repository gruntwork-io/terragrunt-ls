// Package definition provides values-specific go-to-definition functionality.
package definition

import (
	"os"
	"path/filepath"
	"strings"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"

	"go.lsp.dev/protocol"
)

const (
	// DefinitionContextValuesDependency is the context for navigating to a dependency config
	// referenced inside a terragrunt.values.hcl file.
	DefinitionContextValuesDependency = "values_dependency"
)

// GetValuesDefinitionTargetWithContext analyzes the position in a terragrunt.values.hcl file
// and returns navigation information with a classifying context.
func GetValuesDefinitionTargetWithContext(l logger.Logger, s store.Store, position protocol.Position) (string, string) {
	if s.AST == nil {
		l.Debug("No AST found for values file")
		return "", DefinitionContextNull
	}

	word := text.GetCursorWord(s.Document, position)
	if len(word) == 0 {
		l.Debug("No word found at position", "line", position.Line, "character", position.Character)
		return "", DefinitionContextNull
	}

	if strings.Contains(word, ".") {
		parts := strings.Split(word, ".")
		if len(parts) >= 2 && parts[0] == "dependency" {
			depName := parts[1]
			l.Debug("Found dependency reference for definition", "dependency", depName, "word", word)

			return depName, DefinitionContextValuesDependency
		}
	}

	l.Debug("No values-specific definition target found", "word", word)

	return "", DefinitionContextNull
}

// ResolveValuesDependencyPath looks for a dependency unit's terragrunt.hcl in the directory
// structure around the values file, trying common sibling/parent layouts. Returns the first
// match found, or "" if nothing is found.
func ResolveValuesDependencyPath(dependencyName, valuesFile string) (string, bool) {
	currentDir := filepath.Dir(valuesFile)

	candidates := []string{
		filepath.Join(currentDir, "..", dependencyName, "terragrunt.hcl"),
		filepath.Join(currentDir, dependencyName, "terragrunt.hcl"),
		filepath.Join(filepath.Dir(currentDir), dependencyName, "terragrunt.hcl"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}

	return "", false
}
