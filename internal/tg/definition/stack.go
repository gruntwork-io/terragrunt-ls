// Package definition provides stack-specific go-to-definition functionality.
package definition

import (
	"os"
	"path/filepath"
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"

	"go.lsp.dev/protocol"
)

const (
	// DefinitionContextStackSource is the context for navigating to a source location
	DefinitionContextStackSource = "stack_source"

	// DefinitionContextStackPath is the context for navigating to a unit path
	DefinitionContextStackPath = "stack_path"

	// DefinitionContextStackUnit is the context for a unit block
	DefinitionContextStackUnit = "stack_unit"
)

// GetStackDefinitionTargetWithContext analyzes the position in a stack file and returns navigation information
func GetStackDefinitionTargetWithContext(
	l logger.Logger,
	store store.StackStore,
	position protocol.Position,
) (string, string) {
	if store.AST == nil {
		l.Debug("No AST found for stack file")
		return "", DefinitionContextNull
	}

	// Convert LSP position to HCL position
	pos := ast.ToHCLPos(position)
	node := store.AST.FindNodeAt(pos)

	if node == nil {
		l.Debug("No node found at position", "line", position.Line, "character", position.Character)
		return "", DefinitionContextNull
	}

	// Check if we're in a unit block
	if _, ok := store.AST.FindUnitAt(pos); ok {
		// Check if we're hovering over source attribute - navigate to source if local
		if source, ok := store.AST.GetUnitSource(node); ok {
			l.Debug("Found unit source for definition", "source", source)
			return source, DefinitionContextStackSource
		}

		// Check if we're hovering over path attribute - navigate to unit directory
		//
		// FIXME: This is not correct.
		// We actually want to navigate to the directory within `.terragrunt-stack`
		// unless the user has specified `no_dot_stack` in their stack configuration.
		if path, ok := store.AST.GetUnitPath(node); ok {
			l.Debug("Found unit path for definition", "path", path)
			return path, DefinitionContextStackPath
		}
	}

	// Check if we're in a stack block - could navigate to nested stack
	if _, ok := store.AST.FindStackAt(pos); ok {
		// Check if we're hovering over source attribute in stack block
		if source, ok := store.AST.GetUnitSource(node); ok { // reuse GetUnitSource since it's the same structure
			l.Debug("Found stack source for definition", "source", source)
			return source, DefinitionContextStackSource
		}

		// Check if we're hovering over path attribute in stack block
		//
		// FIXME: This is not correct.
		// We actually want to navigate to the directory within `.terragrunt-stack`
		// unless the user has specified `no_dot_stack` in their stack configuration.
		if path, ok := store.AST.GetUnitPath(node); ok { // reuse GetUnitPath since it's the same structure
			l.Debug("Found stack path for definition", "path", path)
			return path, DefinitionContextStackPath
		}
	}

	l.Debug("No stack-specific definition target found")

	return "", DefinitionContextNull
}

// ResolveStackSourceLocation attempts to resolve a source to a local file path
func ResolveStackSourceLocation(source, currentDir string) (string, bool) {
	// Handle relative paths
	if !filepath.IsAbs(source) {
		resolved := filepath.Join(currentDir, source)

		// Check if it's a directory.
		// This implicitly only handles local paths.
		if stat, err := os.Stat(resolved); err == nil {
			if stat.IsDir() {
				return resolved, true
			}
		}
	}

	return "", false
}

// ResolveStackUnitPath resolves a unit path to a terragrunt.hcl file
func ResolveStackUnitPath(path, currentDir string) (string, bool) {
	// Unit paths are relative to the stack directory
	resolved := filepath.Join(currentDir, path, "terragrunt.hcl")

	if _, err := os.Stat(resolved); err == nil {
		return resolved, true
	}

	return "", false
}
