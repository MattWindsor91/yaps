package comm

// File controllable.go contains Controllable, an interface for inner Controller states.

// ResponseCb is the type of response callbacks.
type ResponseCb func(interface{})

// Controllable is the interface for inner Controller states.
type Controllable interface {
	// Dump dumps out the Controllable's public state, calling dumpCb for each dump response.
	Dump(dumpCb ResponseCb)

	// HandleRequest handles a request with body rbody, reply callback replyCb, and broadcast callback bcastCb.
	HandleRequest(replyCb ResponseCb, bcastCb ResponseCb, rbody interface{}) error
}
