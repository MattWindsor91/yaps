package list

// This file contains the responses a list Controller can send.
// See 'controller.go' for the Controller implementation.
// See 'bifrost.go' for a mapping between these and Bifrost messages.

// When adding new responses, make sure to add:
// - controller logic in 'controller.go';
// - a parser from messages in 'bifrost.go';
// - an emitter to messages in 'bifrost.go'.

import "github.com/UniversityRadioYork/baps3d/bifrost"

// Response is the base structure for responses from a Controller.
type Response struct {
	// Broadcast gives whether this is a broadcast response.
	Broadcast bool

	// Origin, if 'Broadcast' is false, gives the original request's RequestOrigin.
	// Else, it is nil.
	Origin *RequestOrigin

	// Body gives the body of the response.
	Body interface{}
}

// Tag gets the correct tag for Response r.
func (r *Response) Tag() string {
	if r.Broadcast {
		return bifrost.TagBcast
	}
	if r.Origin == nil {
		return bifrost.TagUnknown
	}
	if r.Origin.Message == nil {
		return bifrost.TagUnknown
	}

	return r.Origin.Message.Tag()
}

// AckResponse announces that a command has finished processing.
type AckResponse struct {
	// Err, if non-nil, is the error encountered during command processing.
	Err error
}

// AutoModeResponse announces a change in AutoMode.
type AutoModeResponse struct {
	// AutoMode represents the new AutoMode.
	AutoMode AutoMode
}

// ListDumpResponse announces a snapshot of the entire list.
type ListDumpResponse []Item

// ListItemResponse announces the presence of a single list item.
type ListItemResponse struct {
	// Index is the index of the item in the list.
	Index int
	// Item is the item itself.
	Item Item
}
