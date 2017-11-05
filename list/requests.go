package list

// This file contains the requests a list Controller understands.
// See 'controller.go' for the Controller implementation.
// See 'bifrost.go' for a mapping between these and Bifrost messages.

// When adding new responses, make sure to add:
// - controller logic in 'controller.go';
// - a parser from messages in 'bifrost.go';
// - an emitter to messages in 'bifrost.go'.

// RequestOrigin is the structure identifying where a request originated.
type RequestOrigin struct {
	// Tag represents the tag of the request, if applicable.
	Tag string

	// TODO(CaptainHayashi): reply channel
}

// Request is the base structure for requests to a Controller.
type Request struct {
	// Origin gives information about the requester.
	Origin RequestOrigin

	// Body gives the body of the request.
	Body interface{}
}

// SetSelectRequest requests a selection change.
type SetSelectRequest struct {
	// Index represents the index to select.
	Index int
	// Hash represents the hash of the item to select.
	// It exists to prevent selection races.
	Hash string
}

// NextRequest requests a selection skip.
type NextRequest struct {
}

// SetAutoModeRequest requests an automode change.
type SetAutoModeRequest struct {
	// AutoMode represents the new AutoMode to use.
	AutoMode AutoMode
}
