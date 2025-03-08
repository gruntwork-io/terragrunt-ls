package lsp

import "go.lsp.dev/protocol"

type FormatRequest struct {
	Request
	Params protocol.DocumentFormattingParams `json:"params"`
}

type FormatResponse struct {
	Response
	Result []protocol.TextEdit `json:"result"`
}
