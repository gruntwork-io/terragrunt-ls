package lsp

import "go.lsp.dev/protocol"

type DefinitionRequest struct {
	Request
	Params protocol.DefinitionParams `json:"params"`
}

type DefinitionResponse struct {
	Response
	Result protocol.Location `json:"result"`
}
