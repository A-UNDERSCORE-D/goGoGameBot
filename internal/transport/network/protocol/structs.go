// Package protocol holds structs and other constructs that are used by
// the network transport for cross-network RPC calls
package protocol

import "errors"

// SerialiseError is an error struct that is safe to transfer over a gob encoding
type SerialiseError struct {
	ErrorStr string
	IsError  bool
}

// ToError converts a SerialiseError to an error interface
func (s *SerialiseError) ToError() error {
	if s.IsError {
		return errors.New(s.ErrorStr)
	}
	return nil
}

// SErrorFromError converts an error to a SerializeError
func SErrorFromError(err error) SerialiseError {
	if err != nil {
		return SerialiseError{ErrorStr: err.Error(), IsError: true}
	}

	return SerialiseError{}
}

// SErrorFromString creates a SerialiseError from the given string
func SErrorFromString(str string) SerialiseError {
	if str != "" {
		return SerialiseError{ErrorStr: str, IsError: true}
	}

	return SerialiseError{}
}

// StdIOLines holds a set of lines for processing
type StdIOLines struct {
	Lines []StdIOLine
	// Stdout bool
	Error SerialiseError
}

// ProcessExit represents all available information regarding a process that has exited
type ProcessExit struct {
	Return    int
	StrReturn string
	Error     SerialiseError
}

// StdIOLine holds a single line from a process sent over StdIO
type StdIOLine struct {
	Line   string
	Stdout bool
	ID     int64
}
