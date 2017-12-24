package comm

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

// AckResponse announces that a command has finished processing.
type AckResponse struct {
	// Err, if non-nil, is the error encountered during command processing.
	Err error
}

// NewClientResponse responds to a request for a new client connection.
type NewClientResponse struct {
	// Client is the new client connector.
	Client *Client
}

// RoleResponse announces the Controller's Bifrost role.
type RoleResponse struct {
	// Role is the role of the Controller.
	Role string
}
