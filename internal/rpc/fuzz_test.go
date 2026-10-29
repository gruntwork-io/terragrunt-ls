package rpc_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"terragrunt-ls/internal/rpc"
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzSplit tests that Split never panics and never returns more bytes than it was given.
func FuzzSplit(f *testing.F) {
	for _, seed := range []string{
		"Content-Length: 16\r\n\r\n{\"Testing\":true}",
		"Content-Length: 2\r\nContent-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\n{}",
		// Regression: a header shorter than "Content-Length: " panicked.
		"\r\n\r\n",
		"0000\r\n\r\n",
		// Regression: a negative Content-Length panicked.
		"Content-Length: -100\r\n\r\n{}",
	} {
		f.Add([]byte(seed), false)
		f.Add([]byte(seed), true)
	}

	f.Fuzz(func(t *testing.T, data []byte, atEOF bool) {
		advance, token, err := rpc.Split(data, atEOF)
		if err != nil {
			return
		}

		if advance == 0 {
			require.Nil(t, token, "no token until a whole message is buffered")

			return
		}

		require.Positive(t, advance)
		require.LessOrEqual(t, advance, len(data))
		require.Equal(t, data[:advance], token)
	})
}

// FuzzScanMessages tests the stdin read loop in main: Split on a scanner, then DecodeMessage on each token.
func FuzzScanMessages(f *testing.F) {
	f.Add([]byte("Content-Length: 15\r\n\r\n{\"method\":\"hi\"}Content-Length: 15\r\n\r\n{\"method\":\"yo\"}"))
	f.Add([]byte("Content-Length: 2\r\n\r\n{}Content-Length: 1\r\n\r\n["))
	f.Add([]byte("Content-Length: 15\r\nContent-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\n{\"method\":\"hi\"}"))
	// Regression: an empty header panicked.
	f.Add([]byte("\r\n\r\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Split(rpc.Split)

		for scanner.Scan() {
			token := scanner.Bytes()

			method, content, err := rpc.DecodeMessage(token)
			if err != nil {
				continue
			}

			require.True(t, bytes.HasSuffix(token, content), "content must be the end of the token")

			var message struct {
				Method string `json:"method"`
			}

			require.NoError(t, json.Unmarshal(content, &message))
			require.Equal(t, message.Method, method)
		}
	})
}
