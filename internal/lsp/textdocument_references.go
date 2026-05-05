package lsp

import "go.lsp.dev/protocol"

type ReferencesRequest struct {
	Params protocol.ReferenceParams `json:"params"`
	Request
}

type ReferencesResponse struct {
	Response
	Result []protocol.Location `json:"result"`
}
