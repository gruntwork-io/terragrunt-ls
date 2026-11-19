package definition

import (
	"io/fs"
	"path/filepath"
	"terragrunt-ls/internal/ast"
	aststack "terragrunt-ls/internal/ast/stack"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/redact"
	"terragrunt-ls/internal/stackutils"
	"terragrunt-ls/internal/tg/store"

	"github.com/gruntwork-io/terragrunt/pkg/config"
	"go.lsp.dev/protocol"
)

// StatFS is the filesystem subset needed to resolve stack definitions.
type StatFS interface {
	Stat(name string) (fs.FileInfo, error)
}

// StackDefinitionTarget returns the config file that go-to-definition at position in a
// terragrunt.stack.hcl file opens, or "" when there is none. stackDir is the directory
// holding the stack file.
//
// A literal `source` of a top-level unit or stack block opens the terragrunt.hcl or
// terragrunt.stack.hcl in that directory. A `path` of a top-level block opens the
// generated config of the first enabled instance, in declaration order, that
// `terragrunt stack generate` has written. The target must be a regular file.
//
// Panics when s.AST is nil. State.Definition in package tg returns early for a document
// without one.
func StackDefinitionTarget(
	l logger.Logger,
	fsys StatFS,
	s *store.Store,
	position protocol.Position,
	stackDir string,
) string {
	node := s.AST.FindNodeAt(ast.ToHCLPos(position))
	if node == nil {
		l.Debug("No node found at position", "line", position.Line, "character", position.Character)
		return ""
	}

	for _, kind := range []*blockKind{&unitKind, &stackKind} {
		if target := blockDefinition(l, fsys, kind, s.StackCfg, node, stackDir); target != "" {
			return target
		}
	}

	return ""
}

// blockKind describes how to find the definition target of a unit or stack block.
type blockKind struct {
	source     func(*ast.IndexedNode) (string, bool)
	pathLabel  func(*ast.IndexedNode) (string, bool)
	dirs       func(*config.StackConfig, string, string) []string
	name       string
	configFile string
}

var (
	unitKind = blockKind{
		name:       "unit",
		source:     aststack.UnitSource,
		pathLabel:  aststack.UnitPathLabel,
		dirs:       stackutils.UnitDirs,
		configFile: "terragrunt.hcl",
	}

	stackKind = blockKind{
		name:       "stack",
		source:     aststack.StackSource,
		pathLabel:  aststack.StackPathLabel,
		dirs:       stackutils.StackDirs,
		configFile: "terragrunt.stack.hcl",
	}
)

// blockDefinition returns the config file that node navigates to when it sits in the
// `source` or `path` attribute of a block of this kind, or "" otherwise.
func blockDefinition(
	l logger.Logger,
	fsys StatFS,
	kind *blockKind,
	stackCfg *config.StackConfig,
	node *ast.IndexedNode,
	stackDir string,
) string {
	if source, ok := kind.source(node); ok {
		dir := source
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(stackDir, dir)
		}

		target := filepath.Join(dir, kind.configFile)
		if !isRegularFile(fsys, target) {
			l.Debug("No "+kind.configFile+" at "+kind.name+" source", "source", redact.NewURL(source))
			return ""
		}

		return target
	}

	name, ok := kind.pathLabel(node)
	if !ok {
		return ""
	}

	for _, dir := range kind.dirs(stackCfg, name, stackDir) {
		target := filepath.Join(dir, kind.configFile)
		if isRegularFile(fsys, target) {
			return target
		}
	}

	l.Debug("No generated "+kind.configFile+" for "+kind.name, "name", name)

	return ""
}

// isRegularFile reports whether path names a regular file, following symlinks.
func isRegularFile(fsys StatFS, path string) bool {
	info, err := fsys.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
