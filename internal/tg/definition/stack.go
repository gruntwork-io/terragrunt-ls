// Package definition provides stack-specific go-to-definition functionality.
package definition

import (
	"os"
	"path/filepath"
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/stackutils"
	"terragrunt-ls/internal/tg/store"

	"github.com/gruntwork-io/terragrunt/config"
	"go.lsp.dev/protocol"
)

const (
	// DefinitionContextUnitSource is the context for navigating to a unit source location
	DefinitionContextUnitSource = "unit_source"

	// DefinitionContextStackSource is the context for navigating to a stack source location
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
	currentDir string,
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
		// Check if we're jumping to a unit source
		if source, ok := store.AST.GetUnitSource(node); ok {
			l.Debug(
				"Found unit source for definition",
				"source", source,
				"currentDir", currentDir,
			)

			return source, DefinitionContextUnitSource
		}

		// Check if we're jumping to a unit path
		if blockName, ok := store.AST.GetUnitLabel(node); ok {
			if path, ok := stackutils.LookupUnitPath(store.StackCfg, blockName); ok {
				l.Debug(
					"Found unit path for definition from parsed config",
					"blockName", blockName,
					"path", path,
					"currentDir", currentDir,
				)

				return resolveBlockPath(l, store, node, path, currentDir, "unit")
			}
		}
	}

	// Check if we're in a stack block
	if _, ok := store.AST.FindStackAt(pos); ok {
		// Check if we're jumping to a stack source
		if source, ok := store.AST.GetStackSource(node); ok {
			l.Debug(
				"Found stack source for definition",
				"source", source,
				"currentDir", currentDir,
			)

			return source, DefinitionContextStackSource
		}

		// Check if we're jumping to a stack path
		if blockName, ok := store.AST.GetStackLabel(node); ok {
			if path, ok := stackutils.LookupStackPath(store.StackCfg, blockName); ok {
				l.Debug(
					"Found stack path for definition from parsed config",
					"blockName", blockName,
					"path", path,
					"currentDir", currentDir,
				)

				return resolveBlockPath(l, store, node, path, currentDir, "stack")
			}
		}
	}

	l.Debug("No stack-specific definition target found")

	return "", DefinitionContextNull
}

// ResolveUnitSourceLocation attempts to resolve a unit source to a Terraform file
func ResolveUnitSourceLocation(source, currentDir string) string {
	var absPath string

	if filepath.IsAbs(source) {
		absPath = source
	} else {
		absPath = filepath.Join(currentDir, source)
	}

	unitFile := filepath.Join(absPath, "terragrunt.hcl")
	if _, err := os.Stat(unitFile); err == nil {
		return unitFile
	}

	return ""
}

// ResolveStackSourceLocation attempts to resolve a stack source to a terragrunt.stack.hcl file
func ResolveStackSourceLocation(source, currentDir string) string {
	var absPath string

	if filepath.IsAbs(source) {
		absPath = source
	} else {
		absPath = filepath.Join(currentDir, source)
	}

	stackFile := filepath.Join(absPath, "terragrunt.stack.hcl")
	if _, err := os.Stat(stackFile); err == nil {
		return stackFile
	}

	return ""
}

// resolveBlockPath handles path resolution for both unit and stack blocks
func resolveBlockPath(
	l logger.Logger,
	store store.StackStore,
	node *ast.IndexedNode,
	path string,
	currentDir string,
	blockType string,
) (string, string) {
	var blockName string

	var hasName bool

	// Get the block name
	if blockType == "unit" {
		blockName, hasName = store.AST.GetUnitLabel(node)
	} else {
		blockName, hasName = store.AST.GetStackLabel(node)
	}

	if !hasName {
		l.Debug("Could not determine " + blockType + " name for path resolution")
		// Fallback to default behavior
		resolved := filepath.Join(currentDir, ".terragrunt-stack", path, "terragrunt.hcl")
		if _, err := os.Stat(resolved); err == nil {
			return resolved, DefinitionContextStackPath
		}

		return "", DefinitionContextNull
	}

	// Look up the block in the parsed configuration to get noStack setting
	noStack := false
	if store.StackCfg != nil {
		noStack = lookupNoStackConfig(store.StackCfg, blockName, blockType)
	}

	// Resolve the path based on no_dot_terragrunt_stack configuration
	var resolved string
	if noStack {
		resolved = filepath.Join(currentDir, path, "terragrunt.hcl")
	} else {
		resolved = filepath.Join(currentDir, ".terragrunt-stack", path, "terragrunt.hcl")
	}

	if _, err := os.Stat(resolved); err == nil {
		l.Debug("Resolved "+blockType+" path", "path", path, "resolved", resolved, "noStack", noStack)
		return resolved, DefinitionContextStackPath
	}

	l.Debug("Could not resolve "+blockType+" path", "path", path, "resolved", resolved, "noStack", noStack)

	return "", DefinitionContextNull
}

// lookupNoStackConfig looks up the no_dot_terragrunt_stack configuration for a given block
func lookupNoStackConfig(stackCfg *config.StackConfig, blockName, blockType string) bool {
	switch blockType {
	case "unit":
		for _, unit := range stackCfg.Units {
			if unit.Name != blockName {
				continue
			}

			if unit.NoStack != nil {
				return *unit.NoStack
			}

			return false
		}

		return false

	case "stack":
		for _, stack := range stackCfg.Stacks {
			if stack.Name != blockName {
				continue
			}

			if stack.NoStack != nil {
				return *stack.NoStack
			}

			return false
		}

		return false

	default:
		return false
	}
}
