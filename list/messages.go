package list

// Requester is the structure identifying where a request originated.
type Requester struct {
	// Tag represents the tag of the request, if applicable.
	Tag string

	// TODO(CaptainHayashi): reply channel
}

// Request is the base structure for requests to a Controller.
type Request struct {
	// Origin gives information about the requester.
	Origin Requester
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
