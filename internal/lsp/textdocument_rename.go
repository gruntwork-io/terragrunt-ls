package lsp

import "go.lsp.dev/protocol"

type RenameRequest struct {
	Request
	Params protocol.RenameParams `json:"params"`
}

type RenameResponse struct {
	Response
	Result *protocol.WorkspaceEdit `json:"result"`
}
