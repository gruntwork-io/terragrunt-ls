package lsp

import "go.lsp.dev/protocol"

type CompletionRequest struct {
	Params protocol.CompletionParams `json:"params"`
	Request
}

type CompletionResponse struct {
	Response
	Result []protocol.CompletionItem `json:"result"`
}
