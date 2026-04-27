// Package hover provides stack-specific hover functionality.
package hover

import (
	"terragrunt-ls/internal/ast"
	aststack "terragrunt-ls/internal/ast/stack"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/stackutils"
	"terragrunt-ls/internal/tg/store"

	"go.lsp.dev/protocol"
)

const (
	// HoverContextStackUnit is the context for hovering over a unit block.
	HoverContextStackUnit = "stack_unit"

	// HoverContextStackSource is the context for hovering over a source attribute.
	HoverContextStackSource = "stack_source"

	// HoverContextStackPath is the context for hovering over a path attribute.
	HoverContextStackPath = "stack_path"

	// HoverContextStackBlock is the context for hovering over a stack block.
	HoverContextStackBlock = "stack_block"
)

// GetStackHoverTargetWithContext analyzes the position in a terragrunt.stack.hcl file
// and returns hover information with a classifying context.
func GetStackHoverTargetWithContext(l logger.Logger, s store.Store, position protocol.Position) (string, string) {
	if s.AST == nil {
		l.Debug("No AST found for stack file")
		return "", HoverContextNull
	}

	stackAST := aststack.NewStackAST(s.AST)

	pos := ast.ToHCLPos(position)
	node := stackAST.FindNodeAt(pos)

	if node == nil {
		l.Debug("No node found at position", "line", position.Line, "character", position.Character)
		return "", HoverContextNull
	}

	if unitBlock, ok := stackAST.FindUnitAt(pos); ok {
		if source, ok := stackAST.GetUnitSource(node); ok {
			l.Debug("Found unit source hover", "source", source)
			return source, HoverContextStackSource
		}

		if blockName, ok := stackAST.GetUnitLabel(node); ok {
			if path, ok := stackutils.LookupUnitPath(s.StackCfg, blockName); ok {
				l.Debug("Found unit path hover from parsed config", "blockName", blockName, "path", path)
				return path, HoverContextStackPath
			}
		}

		if unitLabel, ok := stackAST.BlockLabel(unitBlock); ok {
			l.Debug("Found unit block (general) hover", "unit", unitLabel)
			return unitLabel, HoverContextStackUnit
		}
	}

	if stackBlock, ok := stackAST.FindStackAt(pos); ok {
		if source, ok := stackAST.GetStackSource(node); ok {
			l.Debug("Found stack source hover", "source", source)
			return source, HoverContextStackSource
		}

		if blockName, ok := stackAST.GetStackLabel(node); ok {
			if path, ok := stackutils.LookupStackPath(s.StackCfg, blockName); ok {
				l.Debug("Found stack path hover from parsed config", "blockName", blockName, "path", path)
				return path, HoverContextStackPath
			}
		}

		if stackLabel, ok := stackAST.BlockLabel(stackBlock); ok {
			l.Debug("Found stack block (general) hover", "stack", stackLabel)
			return stackLabel, HoverContextStackBlock
		}
	}

	l.Debug("No stack-specific hover target found")

	return "", HoverContextNull
}
