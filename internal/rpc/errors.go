package rpc

import "fmt"

// MissingContentLengthError is returned when a message header has no `Content-Length: ` field.
type MissingContentLengthError struct {
	Header string
}

func (err MissingContentLengthError) Error() string {
	return fmt.Sprintf("message header %q must have a field %q followed by the body length in bytes", err.Header, contentLengthPrefix)
}

// NegativeContentLengthError is returned when a message header declares a negative body length.
type NegativeContentLengthError struct {
	ContentLength int
}

func (err NegativeContentLengthError) Error() string {
	return fmt.Sprintf("message header declares a negative Content-Length of %d, expected the body length in bytes", err.ContentLength)
}

// TruncatedBodyError is returned when a message body is shorter than the Content-Length in its header.
type TruncatedBodyError struct {
	ContentLength int
	BodyLength    int
}

func (err TruncatedBodyError) Error() string {
	return fmt.Sprintf("message body has %d bytes but the header declares a Content-Length of %d, expected Content-Length to match the body length in bytes", err.BodyLength, err.ContentLength)
}
