// Package references provides the logic for finding all references of an
// identifier across a Terragrunt module.
package references

import (
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/rename"
	"terragrunt-ls/internal/tg/store"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// GetReferences returns LSP locations for every reference (and optionally the
// declaration) of the renameable symbol at position. Returns nil if the cursor
// is not on a renameable identifier.
func GetReferences(l logger.Logger, st store.Store, position protocol.Position, originFile string, configs map[string]store.Store, includeDeclaration bool) []protocol.Location {
	target := rename.GetRenameTarget(l, st, position)
	if target.Context == rename.RenameContextNull {
		return nil
	}

	occurrences := rename.FindAllOccurrences(l, target, originFile, configs)
	if len(occurrences) == 0 {
		return nil
	}

	locations := make([]protocol.Location, 0, len(occurrences))

	for _, occ := range occurrences {
		if occ.IsDefinition && !includeDeclaration {
			continue
		}

		locations = append(locations, protocol.Location{
			URI:   uri.File(occ.File),
			Range: occ.Range,
		})
	}

	return locations
}
