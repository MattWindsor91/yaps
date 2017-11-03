package list

// This file contains the channel protocol for talking to a Controller.
// See 'controller.go' for the Controller implementation.

// RequestOrigin is the structure identifying where a request originated.
type RequestOrigin struct {
	// Tag represents the tag of the request, if applicable.
	Tag string

	// TODO(CaptainHayashi): reply channel
}

// Requester is the interface for requests to a Controller.
type Requester struct {
	Do func(list *List)
}

// Request is the base structure for requests to a Controller.
type Request struct {
	// Origin gives information about the requester.
	Origin RequestOrigin

	// Body gives the body of the request.
	Body Requester
}

func (r Request) Do(list *List) {
	r.Body.Do(list)
}

// SetSelectRequest requests a selection change.
type SetSelectRequest struct {
	Request

	// Index represents the index to select.
	Index int
	// Hash represents the hash of the item to select.
	// It exists to prevent selection races.
	Hash string
}

// NextRequest requests a selection skip.
type NextRequest struct {
	Request
}

// SetAutoModeRequest requests an automode change.
type SetAutoModeRequest struct {
	Request

	// AutoMode represents the new AutoMode to use.
	AutoMode AutoMode
}
