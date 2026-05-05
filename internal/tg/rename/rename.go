// Package rename provides the logic for renaming identifiers in Terragrunt configurations.
package rename

import (
	"regexp"
	"sort"

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

	// Restrict cursor to the first two steps so that, e.g., `include.root.locals.x`
	// only triggers rename when the cursor is on `include` or `root`.
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
// label of an `include` block.
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
	}

	return "", false
}

// FindAllOccurrences returns every occurrence of target within the given file's
// AST: the declaration site (when present) plus all references. The returned
// slice is sorted by (line, column) for determinism.
func FindAllOccurrences(target RenameTarget, file string, st store.Store) []Occurrence {
	if target.Context == RenameContextNull || st.AST == nil || st.AST.HCLFile == nil {
		return nil
	}

	body, ok := st.AST.HCLFile.Body.(*hclsyntax.Body)
	if !ok || body == nil {
		return nil
	}

	occurrences := definitionOccurrences(target, file, st.AST, body)

	ast.WalkReferences(body, target.Context, target.Name, func(_ *hclsyntax.ScopeTraversalExpr, r hcl.Range) {
		occurrences = append(occurrences, Occurrence{
			File:         file,
			Range:        ast.FromHCLRange(r),
			IsDefinition: false,
		})
	})

	// HCL walks attributes in map iteration order, which is non-deterministic.
	// Sort for stable test output and predictable client-side application.
	sort.Slice(occurrences, func(i, j int) bool {
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

	case RenameContextInclude:
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
