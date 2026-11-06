package lsp

import "go.lsp.dev/protocol"

type PrepareRenameRequest struct {
	Params protocol.PrepareRenameParams `json:"params"`
	Request
}

// PrepareRenameResult mirrors the LSP `{ range, placeholder }` response shape.
type PrepareRenameResult struct {
	Placeholder string         `json:"placeholder"`
	Range       protocol.Range `json:"range"`
}

type PrepareRenameResponse struct {
	Result *PrepareRenameResult `json:"result"`
	Response
}

type RenameRequest struct {
	Params protocol.RenameParams `json:"params"`
	Request
}

type RenameResponse struct {
	Result *protocol.WorkspaceEdit `json:"result"`
	Response
}
