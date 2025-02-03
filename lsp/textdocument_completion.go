package lsp

import "go.lsp.dev/protocol"

type CompletionRequest struct {
	Request
	Params protocol.CompletionParams `json:"params"`
}

type CompletionResponse struct {
	Response
	Result []protocol.CompletionItem `json:"result"`
}
