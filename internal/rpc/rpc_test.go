package rpc_test

import (
	"strconv"
	"terragrunt-ls/internal/rpc"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type EncodingExample struct {
	Testing bool
}

func TestEncode(t *testing.T) {
	t.Parallel()

	expected := "Content-Length: 16\r\n\r\n{\"Testing\":true}"
	actual := rpc.EncodeMessage(EncodingExample{Testing: true})
	if expected != actual {
		t.Fatalf("Expected: %s, Actual: %s", expected, actual)
	}
}

func TestDecode(t *testing.T) {
	t.Parallel()

	incomingMessage := "Content-Length: 15\r\n\r\n{\"Method\":\"hi\"}"
	method, content, err := rpc.DecodeMessage([]byte(incomingMessage))
	contentLength := len(content)
	if err != nil {
		t.Fatal(err)
	}

	if contentLength != 15 {
		t.Fatalf("Expected: 16, Got: %d", contentLength)
	}

	if method != "hi" {
		t.Fatalf("Expected: 'hi', Got: %s", method)
	}
}

func TestSplit(t *testing.T) {
	t.Parallel()

	tc := []struct {
		expectedErr     error
		name            string
		data            string
		expectedToken   string
		expectedAdvance int
	}{
		{
			name:            "complete message",
			data:            "Content-Length: 2\r\n\r\n{}",
			expectedAdvance: 23,
			expectedToken:   "Content-Length: 2\r\n\r\n{}",
		},
		{
			name:            "first of two messages",
			data:            "Content-Length: 2\r\n\r\n{}Content-Length: 2\r\n\r\n[]",
			expectedAdvance: 23,
			expectedToken:   "Content-Length: 2\r\n\r\n{}",
		},
		{
			name: "body not fully read yet",
			data: "Content-Length: 5\r\n\r\n{}",
		},
		{
			name: "header not fully read yet",
			data: "Content-Length: 2\r\n",
		},
		{
			name:        "header shorter than Content-Length",
			data:        "X\r\n\r\n{}",
			expectedErr: rpc.MissingContentLengthError{Header: "X"},
		},
		{
			name:        "empty header",
			data:        "\r\n\r\n",
			expectedErr: rpc.MissingContentLengthError{Header: ""},
		},
		{
			name:        "Content-Length that is not a number",
			data:        "Content-Length: two\r\n\r\n{}",
			expectedErr: strconv.ErrSyntax,
		},
		{
			name:        "negative Content-Length",
			data:        "Content-Length: -100\r\n\r\n{}",
			expectedErr: rpc.NegativeContentLengthError{ContentLength: -100},
		},
		{
			name:        "Content-Length that points inside the header",
			data:        "Content-Length: -4\r\n\r\n{}",
			expectedErr: rpc.NegativeContentLengthError{ContentLength: -4},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			advance, token, err := rpc.Split([]byte(tt.data), false)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedAdvance, advance)
			assert.Equal(t, tt.expectedToken, string(token))
		})
	}
}

func TestDecodeRejectsMalformedHeader(t *testing.T) {
	t.Parallel()

	method, content, err := rpc.DecodeMessage([]byte("X\r\n\r\n{}"))
	require.ErrorIs(t, err, rpc.MissingContentLengthError{Header: "X"})
	assert.Empty(t, method)
	assert.Nil(t, content)
}
