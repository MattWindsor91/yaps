package comm

// This file defines Controller, an object that encapsulates part of a baps3d server's state and provides a Request/Response interface to it.
// The baps3d state must satisfy the 'Controllable' interface.

import (
	"reflect"
)

// Controller wraps a List in a channel-based interface.
type Controller struct {
	// state is the internal state managed by the Controller.
	state Controllable

	// clients is the list of Controller-facing client channel pairs.
	// Each client that subscribes gets a Client struct with the other sides.
	clients map[coclient]struct{}

	// running is the internal is-running flag.
	// When this is set to false, the controller loop will exit.
	running bool
}

// Client is the type of external Controller client handles.
type Client struct {
	// Tx is the channel through which the Client can send requests to the Controller.
	Tx chan<- Request

	// Rx is the channel on which the Controller sends status update messages.
	Rx <-chan Response
}

// coclient is the type of internal client handles.
type coclient struct {
	// tx is the status update send channel.
	tx chan<- Response

	// rx is the request receiver channel.
	rx <-chan Request
}

// makeClient creates a new client and coclient pair.
func makeClient() (Client, coclient) {
	rq := make(chan Request)
	rs := make(chan Response)
	cli := Client{Tx: rq, Rx: rs}
	ccl := coclient{tx: rs, rx: rq}
	return cli, ccl
}

// NewController constructs a new Controller for a given Controllable.
func NewController(c Controllable) (*Controller, *Client) {
	client, co := makeClient()

	coclients := make(map[coclient]struct{})
	coclients[co] = struct{}{}

	controller := Controller{
		state:   c,
		clients: coclients,
	}

	return &controller, &client
}

// Run runs this Controller's event loop.
func (c *Controller) Run() {
	cases := make([]reflect.SelectCase, len(c.clients))
	i := 0
	for cl := range c.clients {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cl.rx)}
		i++
	}

	c.running = true
	for c.running {
		// TODO(@MattWindsor91): recalculate client cases when forking

		_, value, ok := reflect.Select(cases)
		if !ok {
			break
		}
		// TODO(@MattWindsor91): properly handle if this isn't a Request
		rq, ok := value.Interface().(Request)
		if !ok {
			panic("FIXME: got bad request")
		}

		c.handleRequest(rq)
	}

	c.hangupClients()
}

// hangupClients hangs up every connected client.
func (c *Controller) hangupClients() {
	for cl := range c.clients {
		c.hangupClient(cl)
	}
}

// hangupClient closes a client's channels and removes it from the client list.
func (c *Controller) hangupClient(cl coclient) {
	close(cl.tx)
	delete(c.clients, cl)
}

//
// Request handling
//

// handleRequest handles a Request rq.
// If the request is a standard Request, the Controller will handle it itself.
// Otherwise, the Controller forwards it to the Controllable.
func (c *Controller) handleRequest(rq Request) {
	var err error

	o := rq.Origin
	switch body := rq.Body.(type) {
	case RoleRequest:
		err = c.handleRoleRequest(o, body)
	case DumpRequest:
		err = c.handleDumpRequest(o, body)
	case ShutdownRequest:
		err = c.handleShutdownRequest(o, body)
	default:
		replyCb := func(rbody interface{}) {
			c.reply(o, rbody)
		}
		err = c.state.HandleRequest(c.broadcast, replyCb, body)
	}

	ack := AckResponse{err}
	c.reply(o, ack)
}

// handleRoleRequest handles a role request with origin o and body b.
func (c *Controller) handleRoleRequest(o RequestOrigin, b RoleRequest) error {
	c.reply(o, RoleResponse{Role: c.state.RoleName()})

	// Role requests never fail
	return nil
}

// handleDumpRequest handles a dump with origin o and body b.
func (c *Controller) handleDumpRequest(o RequestOrigin, b DumpRequest) error {
	dumpCb := func(rbody interface{}) {
		c.reply(o, rbody)
	}
	c.state.Dump(dumpCb)

	// Dump requests never fail
	return nil
}

// handleShutdownRequest handles a shutdown request with origin o and body b.
func (c *Controller) handleShutdownRequest(o RequestOrigin, b ShutdownRequest) error {
	// We don't do the shutdown here, but instead when we go round the main loop.
	c.running = false
	return nil
}

//
// Responding
//

// reply sends a unicast response with body rbody to the request origin to.
func (c *Controller) reply(to RequestOrigin, rbody interface{}) {
	reply := Response{
		Broadcast: false,
		Origin:    &to,
		Body:      rbody,
	}

	to.ReplyTx <- reply
}

// broadcast sends a broadcast response with body rbody to all clients.
func (c *Controller) broadcast(rbody interface{}) {
	response := Response{
		Broadcast: true,
		Origin:    nil,
		Body:      rbody,
	}

	for cl := range c.clients {
		cl.tx <- response
	}
}
