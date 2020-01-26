package corecmd

import "errors"

// Blame is the enumeration of possible causes for a Bifrost-related error.
// It is used mainly to work out whether to send a WHAT or a FAIL.
type Blame int

const (
	// BlameUnknown is the default 'blame' value, and represents an unknown cause.
	BlameUnknown Blame = iota
	// BlameClient suggests that the client is to blame for an error.
	BlameClient
	// BlameServer suggests that the server is to blame for an error
	BlameServer
)

// String converts a Blame to a human-readable string.
func (b Blame) String() string {
	switch b {
	case BlameClient:
		return "client"
	case BlameServer:
		return "server"
	default:
		return "unknown"
	}
}

// Blameable is the interface of errors that carry information about blame.
type Blameable interface {
	// Blame gets the blame of a blameable error.
	Blame() Blame
}

// ErrorBlame returns err's Blame() if it is Blameable, and BlameUnknown otherwise.
func ErrorBlame(err error) Blame {
	var errb Blameable
	if !errors.As(err, &errb) {
		return BlameUnknown
	}
	return errb.Blame()
}
