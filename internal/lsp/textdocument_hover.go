package lsp

import "go.lsp.dev/protocol"

type HoverRequest struct {
	Params protocol.HoverParams `json:"params"`
	Request
}

// HoverResponse is the response to a textDocument/hover request.
//
// A nil Result is sent as a JSON null, which tells the client there is nothing to show.
type HoverResponse struct {
	Result *HoverResult `json:"result"`
	Response
}

type HoverResult struct {
	Contents protocol.MarkupContent `json:"contents"`
}
