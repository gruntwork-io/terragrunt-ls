package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
	"terragrunt-ls/lsp"
	"terragrunt-ls/rpc"
	"terragrunt-ls/tg"

	"go.lsp.dev/protocol"
)

func main() {
	logfile := os.Getenv("TG_LS_LOG")
	logger := getLogger(logfile)
	logger.Println("Initializing terragrunt-ls")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	// state := analysis.NewState()
	state := tg.NewState()
	writer := os.Stdout

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			logger.Printf("Got an error: %s", err)
			continue
		}

		handleMessage(logger, writer, state, method, contents)
	}
}

func handleMessage(logger *log.Logger, writer io.Writer, state tg.State, method string, contents []byte) {
	logger.Printf("Received msg with method: %s", method)

	switch method {
	case protocol.MethodInitialize:
		var request lsp.InitializeRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Printf("Failed to parse initialize request: %s", err)
		}

		logger.Printf("Connected to: %s %s",
			request.Params.ClientInfo.Name,
			request.Params.ClientInfo.Version)

		msg := lsp.NewInitializeResponse(request.ID)
		writeResponse(writer, msg)

		logger.Print("Initialized")
	case protocol.MethodTextDocumentDidOpen:
		var notification lsp.DidOpenTextDocumentNotification
		if err := json.Unmarshal(contents, &notification); err != nil {
			logger.Printf("Failed to parse didOpen request: %s", err)
		}

		logger.Printf("Opened: %s", notification.Params.TextDocument.URI)

		diagnostics := state.OpenDocument(notification.Params.TextDocument.URI, notification.Params.TextDocument.Text)
		writeResponse(writer, lsp.PublishDiagnosticsNotification{
			Notification: lsp.Notification{
				RPC:    lsp.RPCVersion,
				Method: protocol.MethodTextDocumentPublishDiagnostics,
			},
			Params: protocol.PublishDiagnosticsParams{
				URI:         notification.Params.TextDocument.URI,
				Diagnostics: diagnostics,
			},
		})

		logger.Print(state.Documents)

		logger.Print("Document opened")
	}
}

func writeResponse(writer io.Writer, msg any) {
	reply := rpc.EncodeMessage(msg)
	writer.Write([]byte(reply))

}

func getLogger(filename string) *log.Logger {
	if filename == "" {
		return log.New(os.Stderr, "[terragrunt-ls] ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic("Failed to open log file: " + err.Error())
	}

	return log.New(logfile, "[terragrunt-ls] ", log.Ldate|log.Ltime|log.Lshortfile)
}
