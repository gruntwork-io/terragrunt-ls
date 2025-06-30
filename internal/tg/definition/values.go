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
	DefinitionContextValuesDependency = "values_dependency"
)

// GetValuesDefinitionTargetWithContext analyzes the position in a values file and returns navigation information
func GetValuesDefinitionTargetWithContext(l logger.Logger, store store.ValuesStore, position protocol.Position) (string, string) {
	if store.AST == nil {
		l.Debug("No AST found for values file")
		return "", DefinitionContextNull
	}

	// Get the word at the cursor position
	word := text.GetCursorWord(store.Document, position)
	if len(word) == 0 {
		l.Debug("No word found at position", "line", position.Line, "character", position.Character)
		return "", DefinitionContextNull
	}

	// Check if it's a dependency reference (dependency.name.output)
	if strings.Contains(word, ".") {
		parts := strings.Split(word, ".")
		if len(parts) >= 2 && parts[0] == "dependency" {
			depName := parts[1]
			l.Debug("Found dependency reference for definition", "dependency", depName, "word", word)

			return depName, DefinitionContextValuesDependency
		}
	}

	// For now, we don't handle other types of navigation in values files
	l.Debug("No values-specific definition target found", "word", word)

	return "", DefinitionContextNull
}

// ResolveValuesDependencyPath attempts to find the dependency terragrunt.hcl file
func ResolveValuesDependencyPath(dependencyName, valuesFile string) (string, bool) {
	// Parse the values file to find the dependency block
	// For now, we'll use a simple approach - look for dependency blocks in the directory structure
	currentDir := filepath.Dir(valuesFile)

	// Common patterns:
	// 1. ../dependency-name/terragrunt.hcl
	// 2. dependency-name/terragrunt.hcl
	// 3. Look for any dependency block in the values file to get the config_path

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
