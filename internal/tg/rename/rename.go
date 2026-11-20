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
	// for use as the prepare-rename range.
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
func GetRenameTarget(l logger.Logger, st *store.Store, position protocol.Position) RenameTarget {
	null := RenameTarget{Context: RenameContextNull}

	if st.AST == nil {
		l.Debug("No AST found for rename")
		return null
	}

	offset := ast.ByteOffset(st.Document, position)

	inode := st.AST.FindNodeAt(position)
	if inode == nil {
		l.Debug("No node at position", "line", position.Line, "character", position.Character)
		return null
	}

	if expr, ok := inode.Node.(*hclsyntax.ScopeTraversalExpr); ok {
		return traversalTarget(st.Document, expr, offset)
	}

	for cur := inode; cur != nil; cur = cur.Parent {
		attr, ok := cur.Node.(*hclsyntax.Attribute)
		if !ok {
			continue
		}

		if !ast.IsLocalAttribute(cur) {
			return null
		}

		if !attr.NameRange.ContainsOffset(offset) {
			return null
		}

		return RenameTarget{
			Name:       attr.Name,
			Context:    RenameContextLocal,
			IdentRange: ast.FromHCLRange(st.Document, attr.NameRange),
		}
	}

	return null
}

// traversalTarget extracts a RenameTarget from a ScopeTraversalExpr in doc if offset, the
// cursor's byte offset in doc, is on its first two traversal steps and the root is `local`.
func traversalTarget(doc string, expr *hclsyntax.ScopeTraversalExpr, offset int) RenameTarget {
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

	if rootStep.Name != "local" {
		return null
	}

	// Restrict cursor to the first two steps.
	firstTwo := hcl.Range{
		Filename: rootStep.SrcRange.Filename,
		Start:    rootStep.SrcRange.Start,
		End:      attrStep.SrcRange.End,
	}
	if !firstTwo.ContainsOffset(offset) {
		return null
	}

	return RenameTarget{
		Name:       attrStep.Name,
		Context:    RenameContextLocal,
		IdentRange: ast.FromHCLRange(doc, ast.TraverseAttrIdentRange(&attrStep)),
	}
}

// FindAllOccurrences returns every occurrence of target within the given file's
// AST: the declaration site (when present) plus all references. The returned
// slice is sorted by (line, column) for determinism.
func FindAllOccurrences(target RenameTarget, file string, st *store.Store) []Occurrence {
	if target.Context == RenameContextNull || st.AST == nil || st.AST.HCLFile == nil {
		return nil
	}

	body, ok := st.AST.HCLFile.Body.(*hclsyntax.Body)
	if !ok || body == nil {
		return nil
	}

	occurrences := definitionOccurrences(target, file, st)

	ast.WalkReferences(body, target.Context, target.Name, func(_ *hclsyntax.ScopeTraversalExpr, r hcl.Range) {
		occurrences = append(occurrences, Occurrence{
			File:         file,
			Range:        ast.FromHCLRange(st.Document, r),
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
func definitionOccurrences(target RenameTarget, file string, st *store.Store) []Occurrence {
	if target.Context != RenameContextLocal {
		return nil
	}

	def, ok := st.AST.Locals[target.Name]
	if !ok {
		return nil
	}

	attr, ok := def.Node.(*hclsyntax.Attribute)
	if !ok {
		return nil
	}

	return []Occurrence{{
		File:         file,
		Range:        ast.FromHCLRange(st.Document, attr.NameRange),
		IsDefinition: true,
	}}
}
