package comm

// File request.go contains the high-level Request type, and request bodies common to all Controllers.

// RequestOrigin is the structure identifying where a request originated.
type RequestOrigin struct {
	// Tag is a string used to identify this request, if any.
	Tag string

	// ReplyTx is the channel any unicast responses will be sent down.
	ReplyTx chan<- Response
}

// Request is the base structure for requests to a Controller.
// Each Request has a body, which may or may not be specific to the inner controller state.
type Request struct {
	// Origin gives information about the requester.
	Origin RequestOrigin

	// Body gives the body of the request.
	Body interface{}
}

//
// Standard request bodies
//

// DumpRequest requests an information dump.
type DumpRequest struct{}

// NewClientRequest requests that the Controller add a new client.
// It will result in a NewClientResponse reply with the client connector.
type NewClientRequest struct{}

// RoleRequest requests the Bifrost role of the connected Controller.
// It will result in a RoleResponse reply.
type RoleRequest struct{}

// Shutdown requests a shutdown.
// The Controller will not reply, other than immediately sending an AckResponse.
// The shutdown is complete when the Controller closes this client's response channel.
type ShutdownRequest struct{}
