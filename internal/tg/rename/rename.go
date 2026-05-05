// Package rename provides the logic for renaming identifiers in Terragrunt configurations.
package rename

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"go.lsp.dev/protocol"
)

const (
	// RenameContextLocal is the context for renaming a local variable
	// (`locals { name = ... }` and `local.name` references).
	RenameContextLocal = "local"

	// RenameContextDependency is the context for renaming a dependency block
	// label (`dependency "name" {}` and `dependency.name.outputs.X` references).
	RenameContextDependency = "dependency"

	// RenameContextInclude is the context for renaming an include block label
	// (`include "name" {}` and `include.name.X` references).
	RenameContextInclude = "include"

	// RenameContextNull means the cursor is not on a renameable identifier.
	RenameContextNull = "null"
)

// RenameTarget describes the symbol resolved at the cursor position.
type RenameTarget struct {
	// Name is the current identifier value.
	Name string
	// Context is one of the RenameContext* constants.
	Context string
	// IdentRange is the LSP range covering only the identifier token, suitable
	// for use as the prepare-rename range. For block labels this excludes the
	// surrounding quotes.
	IdentRange protocol.Range
}

// Occurrence is a single text span (in a specific file) of the target symbol.
// IsDefinition is true for the symbol's declaration site, false for references.
type Occurrence struct {
	File         string
	Range        protocol.Range
	IsDefinition bool
}

// skippedFiles lists base names that are part of the same folder but represent
// a different file type and should not be scanned for rename occurrences.
var skippedFiles = map[string]struct{}{
	"terragrunt.stack.hcl":  {},
	"terragrunt.values.hcl": {},
}

// hclIdentifierRE matches a valid HCL identifier (also accepts hyphens, which
// are valid in block labels though not in unquoted variable references).
var hclIdentifierRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

// IsValidIdentifier reports whether s is a valid HCL identifier.
func IsValidIdentifier(s string) bool {
	return hclIdentifierRE.MatchString(s)
}

// GetRenameTarget identifies the renameable symbol at the given position.
// Returns a target with Context == RenameContextNull when nothing renameable
// is at the position.
func GetRenameTarget(l logger.Logger, st store.Store, position protocol.Position) RenameTarget {
	null := RenameTarget{Context: RenameContextNull}

	if st.AST == nil {
		l.Debug("No AST found for rename")
		return null
	}

	inode := st.AST.FindNodeAt(ast.ToHCLPos(position))
	if inode == nil {
		l.Debug("No node at position", "line", position.Line, "character", position.Character)
		return null
	}

	if expr, ok := inode.Node.(*hclsyntax.ScopeTraversalExpr); ok {
		return traversalTarget(expr, position)
	}

	for cur := inode; cur != nil; cur = cur.Parent {
		switch n := cur.Node.(type) {
		case *hclsyntax.Block:
			if t, ok := blockLabelTarget(n, position); ok {
				return t
			}
			// We hit an enclosing block but the cursor isn't on its label.
			return null

		case *hclsyntax.Attribute:
			if !ast.IsLocalAttribute(cur) {
				return null
			}

			if !rangeContainsPosition(n.NameRange, position) {
				return null
			}

			return RenameTarget{
				Name:       n.Name,
				Context:    RenameContextLocal,
				IdentRange: ast.FromHCLRange(n.NameRange),
			}
		}
	}

	return null
}

// traversalTarget extracts a RenameTarget from a ScopeTraversalExpr if the
// cursor is positioned on its first two traversal steps and the root is a
// supported kind.
func traversalTarget(expr *hclsyntax.ScopeTraversalExpr, position protocol.Position) RenameTarget {
	null := RenameTarget{Context: RenameContextNull}

	if len(expr.Traversal) < ast.MinReferenceTraversalLen {
		return null
	}

	rootStep, ok := expr.Traversal[0].(hcl.TraverseRoot)
	if !ok {
		return null
	}

	attrStep, ok := expr.Traversal[1].(hcl.TraverseAttr)
	if !ok {
		return null
	}

	// Restrict cursor to the first two steps so that, e.g., `dependency.vpc.outputs.x`
	// only triggers rename when the cursor is on `dependency` or `vpc`.
	firstTwo := hcl.Range{
		Filename: rootStep.SrcRange.Filename,
		Start:    rootStep.SrcRange.Start,
		End:      attrStep.SrcRange.End,
	}
	if !rangeContainsPosition(firstTwo, position) {
		return null
	}

	context, ok := contextForRoot(rootStep.Name)
	if !ok {
		return null
	}

	return RenameTarget{
		Name:       attrStep.Name,
		Context:    context,
		IdentRange: ast.FromHCLRange(ast.TraverseAttrIdentRange(attrStep)),
	}
}

// blockLabelTarget returns a RenameTarget when the cursor is on the first
// label of an `include` or `dependency` block.
func blockLabelTarget(block *hclsyntax.Block, position protocol.Position) (RenameTarget, bool) {
	null := RenameTarget{Context: RenameContextNull}

	context, ok := contextForRoot(block.Type)
	if !ok || context == RenameContextLocal {
		return null, false
	}

	if len(block.Labels) == 0 || len(block.LabelRanges) == 0 {
		return null, false
	}

	if !rangeContainsPosition(block.LabelRanges[0], position) {
		return null, false
	}

	return RenameTarget{
		Name:       block.Labels[0],
		Context:    context,
		IdentRange: labelInnerRange(block.LabelRanges[0]),
	}, true
}

// contextForRoot maps an HCL root identifier (the first traversal step or a
// block type) to a rename context. Returns false for unsupported names.
func contextForRoot(name string) (string, bool) {
	switch name {
	case "local":
		return RenameContextLocal, true
	case "include":
		return RenameContextInclude, true
	case "dependency":
		return RenameContextDependency, true
	}

	return "", false
}

// FindAllOccurrences returns every rename occurrence of target across all
// sibling .hcl files in the same directory as originFile (including originFile
// itself). When a file has an entry in configs with a parsed AST, that AST is
// used; otherwise the file is read from disk and parsed. Files that fail to
// read or parse are skipped.
func FindAllOccurrences(l logger.Logger, target RenameTarget, originFile string, configs map[string]store.Store) []Occurrence {
	if target.Context == RenameContextNull {
		return nil
	}

	files, err := siblingHCLFiles(filepath.Dir(originFile), configs)
	if err != nil {
		l.Error("Failed to list sibling HCL files", "dir", filepath.Dir(originFile), "error", err)
		return nil
	}

	var occurrences []Occurrence

	for _, file := range files {
		iast := getOrParseAST(file, configs, l)
		if iast == nil || iast.HCLFile == nil {
			continue
		}

		body, ok := iast.HCLFile.Body.(*hclsyntax.Body)
		if !ok || body == nil {
			continue
		}

		occurrences = append(occurrences, definitionOccurrences(target, file, iast, body)...)

		ast.WalkReferences(body, target.Context, target.Name, func(_ *hclsyntax.ScopeTraversalExpr, r hcl.Range) {
			occurrences = append(occurrences, Occurrence{
				File:         file,
				Range:        ast.FromHCLRange(r),
				IsDefinition: false,
			})
		})
	}

	// HCL walks attributes in map iteration order, which is non-deterministic.
	// Sort for stable test output and predictable client-side application.
	sort.Slice(occurrences, func(i, j int) bool {
		if occurrences[i].File != occurrences[j].File {
			return occurrences[i].File < occurrences[j].File
		}

		if occurrences[i].Range.Start.Line != occurrences[j].Range.Start.Line {
			return occurrences[i].Range.Start.Line < occurrences[j].Range.Start.Line
		}

		return occurrences[i].Range.Start.Character < occurrences[j].Range.Start.Character
	})

	return occurrences
}

// definitionOccurrences finds the definition site(s) of target in the given file.
func definitionOccurrences(target RenameTarget, file string, iast *ast.IndexedAST, body *hclsyntax.Body) []Occurrence {
	var occs []Occurrence

	switch target.Context {
	case RenameContextLocal:
		def, ok := iast.Locals[target.Name]
		if !ok {
			return nil
		}

		attr, ok := def.Node.(*hclsyntax.Attribute)
		if !ok {
			return nil
		}

		occs = append(occs, Occurrence{
			File:         file,
			Range:        ast.FromHCLRange(attr.NameRange),
			IsDefinition: true,
		})

	case RenameContextInclude, RenameContextDependency:
		for _, blk := range body.Blocks {
			if blk.Type != target.Context {
				continue
			}

			if len(blk.Labels) == 0 || blk.Labels[0] != target.Name {
				continue
			}

			if len(blk.LabelRanges) == 0 {
				continue
			}

			occs = append(occs, Occurrence{
				File:         file,
				Range:        labelInnerRange(blk.LabelRanges[0]),
				IsDefinition: true,
			})
		}
	}

	return occs
}

// siblingHCLFiles returns absolute paths of *.hcl files in dir, excluding
// stack and values files. Files present in configs (open in the editor but
// possibly not yet saved to disk) are included even if they don't exist on disk.
func siblingHCLFiles(dir string, configs map[string]store.Store) ([]string, error) {
	seen := map[string]struct{}{}

	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		if !isRenameableHCLFile(name) {
			continue
		}

		seen[filepath.Join(dir, name)] = struct{}{}
	}

	for path := range configs {
		if filepath.Dir(path) != dir {
			continue
		}

		if !isRenameableHCLFile(filepath.Base(path)) {
			continue
		}

		seen[path] = struct{}{}
	}

	files := make([]string, 0, len(seen))
	for f := range seen {
		files = append(files, f)
	}

	return files, nil
}

// isRenameableHCLFile returns true if the base name is a .hcl file that should
// be scanned for rename occurrences (excludes stack and values files).
func isRenameableHCLFile(base string) bool {
	if !strings.HasSuffix(base, ".hcl") {
		return false
	}

	_, skip := skippedFiles[base]

	return !skip
}

// getOrParseAST returns the parsed IndexedAST for path, preferring an
// already-parsed in-memory AST (so unsaved editor edits are honored).
func getOrParseAST(path string, configs map[string]store.Store, l logger.Logger) *ast.IndexedAST {
	if st, ok := configs[path]; ok && st.AST != nil {
		return st.AST
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		l.Debug("Skipping file (read error)", "file", path, "error", err)
		return nil
	}

	iast, _ := ast.ParseHCLFile(path, contents)

	return iast
}

// labelInnerRange returns the LSP range of a quoted label's contents, excluding
// the surrounding double quotes.
func labelInnerRange(r hcl.Range) protocol.Range {
	r.Start.Column++
	r.Start.Byte++
	r.End.Column--
	r.End.Byte--

	return ast.FromHCLRange(r)
}

// rangeContainsPosition reports whether p (LSP coordinates) is inside r (HCL
// coordinates). The end of an HCL range is exclusive.
func rangeContainsPosition(r hcl.Range, p protocol.Position) bool {
	line := int(p.Line) + 1
	col := int(p.Character) + 1

	if line < r.Start.Line || line > r.End.Line {
		return false
	}

	if line == r.Start.Line && col < r.Start.Column {
		return false
	}

	if line == r.End.Line && col >= r.End.Column {
		return false
	}

	return true
}
