package lsp

type Request struct {
	RPC    string `json:"jsonrpc"`
	Method string `json:"method"`
	ID     int    `json:"id"`
}

type Response struct {
	ID  *int   `json:"id,omitempty"`
	RPC string `json:"jsonrpc"`
}

type Notification struct {
	RPC    string `json:"jsonrpc"`
	Method string `json:"method"`
}
