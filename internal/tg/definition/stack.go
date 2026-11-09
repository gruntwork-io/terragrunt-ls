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

	// DefinitionContextUnitPath is the context for navigating to a unit's resolved path.
	DefinitionContextUnitPath = "unit_path"

	// DefinitionContextStackPath is the context for navigating to a stack's resolved path.
	DefinitionContextStackPath = "stack_path"
)

// GetStackDefinitionTargetWithContext analyzes the position in a terragrunt.stack.hcl file
// and returns navigation information with a classifying context.
func GetStackDefinitionTargetWithContext(
	l logger.Logger,
	s *store.Store,
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

	for _, kind := range []*blockKind{&unitKind, &stackKind} {
		if target, context, ok := blockDefinition(l, kind, stackAST, s.StackCfg, node, currentDir); ok {
			return target, context
		}
	}

	l.Debug("No stack-specific definition target found")

	return "", DefinitionContextNull
}

// blockKind describes how to extract definition information for a unit or stack block.
type blockKind struct {
	getSource     func(*aststack.StackAST, *ast.IndexedNode) (string, bool)
	isPath        func(*aststack.StackAST, *ast.IndexedNode) bool
	getLabel      func(*aststack.StackAST, *ast.IndexedNode) (string, bool)
	lookupDir     func(*config.StackConfig, string, string) (string, bool)
	name          string
	sourceContext string
	pathContext   string
}

var (
	unitKind = blockKind{
		name:          "unit",
		getSource:     (*aststack.StackAST).GetUnitSource,
		isPath:        (*aststack.StackAST).IsUnitPathAttribute,
		getLabel:      (*aststack.StackAST).GetUnitLabel,
		lookupDir:     stackutils.LookupUnitDir,
		sourceContext: DefinitionContextUnitSource,
		pathContext:   DefinitionContextUnitPath,
	}

	stackKind = blockKind{
		name:          "stack",
		getSource:     (*aststack.StackAST).GetStackSource,
		isPath:        (*aststack.StackAST).IsStackPathAttribute,
		getLabel:      (*aststack.StackAST).GetStackLabel,
		lookupDir:     stackutils.LookupStackDir,
		sourceContext: DefinitionContextStackSource,
		pathContext:   DefinitionContextStackPath,
	}
)

// blockDefinition returns the definition target for a node inside a unit or stack block.
// The final return value is false when the node holds nothing navigable for this block kind.
func blockDefinition(
	l logger.Logger,
	kind *blockKind,
	stackAST *aststack.StackAST,
	stackCfg *config.StackConfig,
	node *ast.IndexedNode,
	currentDir string,
) (string, string, bool) {
	if source, ok := kind.getSource(stackAST, node); ok {
		l.Debug("Found "+kind.name+" source for definition", "source", source, "currentDir", currentDir)
		return source, kind.sourceContext, true
	}

	// Only navigate to the generated path when the cursor is on the `path` attribute.
	if !kind.isPath(stackAST, node) {
		return "", "", false
	}

	blockName, ok := kind.getLabel(stackAST, node)
	if !ok {
		return "", "", false
	}

	dir, ok := kind.lookupDir(stackCfg, blockName, currentDir)
	if !ok {
		return "", "", false
	}

	l.Debug("Found "+kind.name+" path for definition from parsed config", "blockName", blockName, "dir", dir)

	target := filepath.Join(dir, "terragrunt.hcl")
	if _, err := os.Stat(target); err != nil {
		l.Debug("Generated "+kind.name+" config does not exist", "target", target)
		return "", DefinitionContextNull, true
	}

	return target, kind.pathContext, true
}

// ResolveUnitSourceLocation resolves a unit `source` to the terragrunt.hcl inside that directory,
// falling back to the directory itself when the file is missing.
// Returns "" if the directory does not exist.
func ResolveUnitSourceLocation(source, currentDir string) string {
	return resolveSourceLocation(source, currentDir, "terragrunt.hcl")
}

// ResolveStackSourceLocation resolves a stack `source` to the terragrunt.stack.hcl
// inside that directory, falling back to the directory itself when the file is missing.
// Returns "" if the directory does not exist.
func ResolveStackSourceLocation(source, currentDir string) string {
	return resolveSourceLocation(source, currentDir, "terragrunt.stack.hcl")
}

func resolveSourceLocation(source, currentDir, fileName string) string {
	absPath := source
	if !filepath.IsAbs(source) {
		absPath = filepath.Join(currentDir, source)
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return ""
	}

	file := filepath.Join(absPath, fileName)
	if _, err := os.Stat(file); err == nil {
		return file
	}

	return absPath
}
