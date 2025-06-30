// Package hover provides stack-specific hover functionality.
package hover

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/stackutils"
	"terragrunt-ls/internal/tg/store"

	"go.lsp.dev/protocol"
)

const (
	// HoverContextStackUnit is the context for hovering over a unit block
	HoverContextStackUnit = "stack_unit"

	// HoverContextStackSource is the context for hovering over a source attribute
	HoverContextStackSource = "stack_source"

	// HoverContextStackPath is the context for hovering over a path attribute
	HoverContextStackPath = "stack_path"

	// HoverContextStackBlock is the context for hovering over a stack block
	HoverContextStackBlock = "stack_block"
)

// GetStackHoverTargetWithContext analyzes the position in a stack file and returns hover information
func GetStackHoverTargetWithContext(l logger.Logger, store store.StackStore, position protocol.Position) (string, string) {
	if store.AST == nil {
		l.Debug("No AST found for stack file")
		return "", HoverContextNull
	}

	// Convert LSP position to HCL position
	pos := ast.ToHCLPos(position)
	node := store.AST.FindNodeAt(pos)

	if node == nil {
		l.Debug("No node found at position", "line", position.Line, "character", position.Character)
		return "", HoverContextNull
	}

	// Check if we're in a unit block
	if _, ok := store.AST.FindUnitAt(pos); ok {
		// Check if we're hovering over source attribute
		if source, ok := store.AST.GetUnitSource(node); ok {
			l.Debug("Found unit source hover", "source", source)
			return source, HoverContextStackSource
		}

		// Check if we're on a path attribute
		if blockName, ok := store.AST.GetUnitLabel(node); ok {
			if path, ok := stackutils.LookupUnitPath(store.StackCfg, blockName); ok {
				l.Debug("Found unit path hover from parsed config", "blockName", blockName, "path", path)
				return path, HoverContextStackPath
			}
		}

		// Fallback: if we're in a unit block but not on a specific attribute, show unit info
		if unitBlock, ok := store.AST.FindUnitAt(pos); ok {
			if unitLabel, ok := store.AST.GetUnitLabel(unitBlock); ok {
				l.Debug("Found unit block (general) hover", "unit", unitLabel)
				return unitLabel, HoverContextStackUnit
			}
		}
	}

	// Check if we're in a stack block
	if _, ok := store.AST.FindStackAt(pos); ok {
		// Check if we're hovering over source attribute
		if source, ok := store.AST.GetStackSource(node); ok {
			l.Debug("Found stack source hover", "source", source)
			return source, HoverContextStackSource
		}

		// Check if we're on a path attribute
		if blockName, ok := store.AST.GetStackLabel(node); ok {
			if path, ok := stackutils.LookupStackPath(store.StackCfg, blockName); ok {
				l.Debug("Found stack path hover from parsed config", "blockName", blockName, "path", path)
				return path, HoverContextStackPath
			}
		}

		// Fallback: if we're in a stack block but not on a specific attribute, show stack info
		if stackBlock, ok := store.AST.FindStackAt(pos); ok {
			if stackLabel, ok := store.AST.GetStackLabel(stackBlock); ok {
				l.Debug("Found stack block (general) hover", "stack", stackLabel)
				return stackLabel, HoverContextStackBlock
			}
		}
	}

	l.Debug("No stack-specific hover target found")

	return "", HoverContextNull
}
