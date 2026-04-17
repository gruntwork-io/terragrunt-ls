// Package definition provides stack-specific go-to-definition functionality.
package definition

import (
	"os"
	"path/filepath"
	"terragrunt-ls/internal/ast"
	aststack "terragrunt-ls/internal/ast/stack"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/stackutils"
	"terragrunt-ls/internal/tg/store"

	"github.com/gruntwork-io/terragrunt/pkg/config"
	"go.lsp.dev/protocol"
)

const (
	// DefinitionContextUnitSource is the context for navigating to a unit source location.
	DefinitionContextUnitSource = "unit_source"

	// DefinitionContextStackSource is the context for navigating to a stack source location.
	DefinitionContextStackSource = "stack_source"

	// DefinitionContextStackPath is the context for navigating to a resolved block path.
	DefinitionContextStackPath = "stack_path"

	// DefinitionContextStackUnit is the context for a unit block.
	DefinitionContextStackUnit = "stack_unit"
)

// GetStackDefinitionTargetWithContext analyzes the position in a terragrunt.stack.hcl file
// and returns navigation information with a classifying context.
func GetStackDefinitionTargetWithContext(
	l logger.Logger,
	s store.Store,
	position protocol.Position,
	currentDir string,
) (string, string) {
	if s.AST == nil {
		l.Debug("No AST found for stack file")
		return "", DefinitionContextNull
	}

	stackAST := aststack.NewStackAST(s.AST)

	pos := ast.ToHCLPos(position)
	node := stackAST.FindNodeAt(pos)

	if node == nil {
		l.Debug("No node found at position", "line", position.Line, "character", position.Character)
		return "", DefinitionContextNull
	}

	if _, ok := stackAST.FindUnitAt(pos); ok {
		if source, ok := stackAST.GetUnitSource(node); ok {
			l.Debug(
				"Found unit source for definition",
				"source", source,
				"currentDir", currentDir,
			)

			return source, DefinitionContextUnitSource
		}

		if blockName, ok := stackAST.GetUnitLabel(node); ok {
			if path, ok := stackutils.LookupUnitPath(s.StackCfg, blockName); ok {
				l.Debug(
					"Found unit path for definition from parsed config",
					"blockName", blockName,
					"path", path,
					"currentDir", currentDir,
				)

				return resolveBlockPath(l, stackAST, s.StackCfg, node, path, currentDir, "unit")
			}
		}
	}

	if _, ok := stackAST.FindStackAt(pos); ok {
		if source, ok := stackAST.GetStackSource(node); ok {
			l.Debug(
				"Found stack source for definition",
				"source", source,
				"currentDir", currentDir,
			)

			return source, DefinitionContextStackSource
		}

		if blockName, ok := stackAST.GetStackLabel(node); ok {
			if path, ok := stackutils.LookupStackPath(s.StackCfg, blockName); ok {
				l.Debug(
					"Found stack path for definition from parsed config",
					"blockName", blockName,
					"path", path,
					"currentDir", currentDir,
				)

				return resolveBlockPath(l, stackAST, s.StackCfg, node, path, currentDir, "stack")
			}
		}
	}

	l.Debug("No stack-specific definition target found")

	return "", DefinitionContextNull
}

// ResolveUnitSourceLocation resolves a unit `source` to a Terraform file to open:
// main.tf if present, else the first *.tf file, else the source directory itself.
// Returns "" if the source directory does not exist.
func ResolveUnitSourceLocation(source, currentDir string) string {
	absPath := source
	if !filepath.IsAbs(source) {
		absPath = filepath.Join(currentDir, source)
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return ""
	}

	mainTF := filepath.Join(absPath, "main.tf")
	if _, err := os.Stat(mainTF); err == nil {
		return mainTF
	}

	entries, err := os.ReadDir(absPath)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if filepath.Ext(entry.Name()) == ".tf" {
				return filepath.Join(absPath, entry.Name())
			}
		}
	}

	return absPath
}

// ResolveStackSourceLocation resolves a stack `source` to the terragrunt.stack.hcl
// inside that directory, falling back to the directory itself when the file is missing.
// Returns "" if the directory does not exist.
func ResolveStackSourceLocation(source, currentDir string) string {
	absPath := source
	if !filepath.IsAbs(source) {
		absPath = filepath.Join(currentDir, source)
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return ""
	}

	stackFile := filepath.Join(absPath, "terragrunt.stack.hcl")
	if _, err := os.Stat(stackFile); err == nil {
		return stackFile
	}

	return absPath
}

// resolveBlockPath resolves a unit or stack block's `path` to the generated terragrunt.hcl,
// honoring no_dot_terragrunt_stack when set on the matching block in the parsed config.
func resolveBlockPath(
	l logger.Logger,
	stackAST aststack.StackAST,
	stackCfg *config.StackConfig,
	node *ast.IndexedNode,
	path string,
	currentDir string,
	blockType string,
) (string, string) {
	var blockName string

	var hasName bool

	if blockType == "unit" {
		blockName, hasName = stackAST.GetUnitLabel(node)
	} else {
		blockName, hasName = stackAST.GetStackLabel(node)
	}

	if !hasName {
		l.Debug("Could not determine " + blockType + " name for path resolution")

		resolved := filepath.Join(currentDir, ".terragrunt-stack", path, "terragrunt.hcl")
		if _, err := os.Stat(resolved); err == nil {
			return resolved, DefinitionContextStackPath
		}

		return "", DefinitionContextNull
	}

	noStack := false
	if stackCfg != nil {
		noStack = lookupNoStackConfig(stackCfg, blockName, blockType)
	}

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

// lookupNoStackConfig reports the no_dot_terragrunt_stack setting for the named block in the
// parsed stack config, or false if the block is missing or the setting is unset.
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
