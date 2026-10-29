package rpc

import "fmt"

// MissingContentLengthError is returned when a message header does not start with `Content-Length: `.
type MissingContentLengthError struct {
	Header string
}

func (err MissingContentLengthError) Error() string {
	return fmt.Sprintf("message header %q must start with %q followed by the body length in bytes", err.Header, contentLengthPrefix)
}

// NegativeContentLengthError is returned when a message header declares a negative body length.
type NegativeContentLengthError struct {
	ContentLength int
}

func (err NegativeContentLengthError) Error() string {
	return fmt.Sprintf("message header declares a negative Content-Length of %d, expected the body length in bytes", err.ContentLength)
}
