package corecmd

import (
	"fmt"
	"strings"
)

// Status is the enumeration of acknowledgement status codes.
type Status int

const (
	// StatusUnknown is the zero value of Status, and represents a failure to work out what the Status actually is.
	// This prevents broken status checks from returning values that could be construed as actual statuses.
	StatusUnknown Status = iota

	// StatusOk represents the OK status:
	// 'command was processed successfully'.
	StatusOk

	// StatusWhat represents the WHAT status:
	// 'command was invalid and could not be processed'.
	StatusWhat

	// StatusFail represents the FAIL status:
	// 'command was valid, but an error occurred when the server tried to process it'.
	StatusFail

	// WordOk is the string equivalent of StatusOk.
	WordOk = "OK"

	// WordOk is the string equivalent of StatusOk.
	WordWhat = "WHAT"

	// WordOk is the string equivalent of StatusOk.
	WordFail = "FAIL"
)

// String converts an Status to its string representation.
func (s Status) String() string {
	switch s {
	case StatusOk:
		return WordOk
	case StatusWhat:
		return WordWhat
	case StatusFail:
		return WordFail
	}
	return fmt.Sprintf("unknown Status: %d", s)
}

// BadStatusError is the type of errors concerning bad status words.
// They directly wrap the received bad word.
type BadStatusError string

// Error implements the error protocol for BadStatusError.
func (b BadStatusError) Error() string {
	return fmt.Sprintf("bad status: %q", string(b))
}

// Blame implements blaming for BadStatusError.
func (b BadStatusError) Blame() Blame {
	return BlameClient
}

// ParseStatus parses the string s as an Status.
// It fails with an error if the status is unknown.
func ParseStatus(s string) (Status, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case WordOk:
		return StatusOk, nil
	case WordWhat:
		return StatusWhat, nil
	case WordFail:
		return StatusFail, nil
	default:
		return StatusUnknown, BadStatusError(s)
	}
}

// ErrorStatus gets the most appropriate status for the error err.
func ErrorStatus(err error) Status {
	switch ErrorBlame(err) {
	case BlameClient:
		return StatusWhat
	case BlameServer:
		return StatusFail
	default:
		// If in doubt, blame the server;
		// these sorts of unknown blame probably originate from an internal error there.
		return StatusFail
	}
}
