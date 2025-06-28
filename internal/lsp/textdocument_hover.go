package lsp

import "go.lsp.dev/protocol"

type HoverRequest struct {
	Params protocol.HoverParams `json:"params"`
	Request
}

type HoverResponse struct {
	Response
	Result HoverResult `json:"result"`
}

type HoverResult struct {
	Contents protocol.MarkupContent `json:"contents"`
}
