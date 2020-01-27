package controller

import "github.com/UniversityRadioYork/baps3d/bifrost"

// File response.go contains the high-level Response type, and response bodies common to all Controllers.

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

//
// Standard response bodies
//

// DoneResponse announces that a command has finished processing.
type DoneResponse struct {
	// Err, if non-nil, is the error encountered during command processing.
	Err error
}

// OnResponse represents a response to a forwarded request.
type OnResponse struct {
	// The string identifier of the mount point from which the request has been forwarded.
	MountPoint string
	// The body of the response being forwarded.
	Request Response
}

//
// Internal response bodies
//

// newClientResponse responds to a request for a new client connection.
type newClientResponse struct {
	// Client is the new client connector.
	Client *Client
}

// bifrostParserResponse responds to a request for a Bifrost parser.
type bifrostParserResponse bifrost.Parser
