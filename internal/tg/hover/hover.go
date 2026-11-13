// Package hover provides the logic for determining the target of a hover.
package hover

import (
	"strings"
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/docs"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"go.lsp.dev/protocol"
)

const (
	// HoverContextLocal is the context for a local hover.
	// This means that a hover is happening on top of a local variable.
	HoverContextLocal = "local"

	// HoverContextNull is the context for a null hover.
	// This means that a hover is happening on top of nothing useful.
	HoverContextNull = "null"
)

func GetHoverTargetWithContext(l logger.Logger, store *store.Store, position protocol.Position) (string, string) {
	word := text.GetCursorWord(store.Document, position)
	if len(word) == 0 {
		l.Debug(
			"No word found",
			"line", position.Line,
			"character", position.Character,
		)

		return word, HoverContextNull
	}

	splitExpression := strings.Split(word, ".")

	const localPartsLen = 2

	if len(splitExpression) != localPartsLen {
		l.Debug(
			"Invalid word found",
			"line", position.Line,
			"character", position.Character,
			"word", word,
		)

		return word, HoverContextNull
	}

	if splitExpression[0] == "local" {
		l.Debug(
			"Found local variable",
			"line", position.Line,
			"character", position.Character,
			"local", splitExpression[1],
		)

		return splitExpression[1], HoverContextLocal
	}

	return word, HoverContextNull
}

// GetDocsTarget returns the kind and name of the HCL reference item under the
// cursor: the name of a function call, or the type of a top-level block or the
// name of a top-level attribute.
func GetDocsTarget(st *store.Store, position protocol.Position) (docs.Kind, string, bool) {
	if st.AST == nil {
		return "", "", false
	}

	pos := ast.ToHCLPos(position)

	for n := st.AST.FindNodeAt(pos); n != nil; n = n.Parent {
		switch node := n.Node.(type) {
		case *hclsyntax.FunctionCallExpr:
			if contains(node.NameRange, pos) {
				return docs.KindFunction, node.Name, true
			}
		case *hclsyntax.Block:
			if contains(node.TypeRange, pos) && isTopLevel(n) {
				return docs.KindBlock, node.Type, true
			}
		case *hclsyntax.Attribute:
			if contains(node.NameRange, pos) && isTopLevel(n) {
				return docs.KindAttribute, node.Name, true
			}
		}
	}

	return "", "", false
}

// contains reports whether pos is in r, comparing lines and columns, since
// positions from the client carry no byte offset.
func contains(r hcl.Range, pos hcl.Pos) bool {
	afterStart := pos.Line > r.Start.Line || pos.Line == r.Start.Line && pos.Column >= r.Start.Column
	beforeEnd := pos.Line < r.End.Line || pos.Line == r.End.Line && pos.Column < r.End.Column

	return afterStart && beforeEnd
}

// isTopLevel reports whether a block or attribute is in the file's body rather
// than in a block. Its parent is the body's list of attributes or blocks, whose
// parent is the body, and the file's body has no parent.
func isTopLevel(n *ast.IndexedNode) bool {
	return n.Parent != nil && n.Parent.Parent != nil && n.Parent.Parent.Parent == nil
}
