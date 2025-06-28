package lsp

import "go.lsp.dev/protocol"

type FormatRequest struct {
	Params protocol.DocumentFormattingParams `json:"params"`
	Request
}

type FormatResponse struct {
	Response
	Result []protocol.TextEdit `json:"result"`
}
