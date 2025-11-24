package lsp

import "go.lsp.dev/protocol"

type PrepareRenameRequest struct {
	Request
	Params protocol.PrepareRenameParams `json:"params"`
}

// PrepareRenameResponse can return:
// - A Range with a placeholder
// - null if rename is not valid at the position
type PrepareRenameResponse struct {
	Response
	Result *PrepareRenameResult `json:"result"`
}

// PrepareRenameResult contains the range and placeholder for the rename operation
type PrepareRenameResult struct {
	Range       protocol.Range `json:"range"`
	Placeholder string         `json:"placeholder"`
}
