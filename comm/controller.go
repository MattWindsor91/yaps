package comm

// This file defines Controller, an object that encapsulates part of a baps3d server's state and provides a Request/Response interface to it.
// The baps3d state must satisfy the 'Controllable' interface.

import (
	"fmt"
	"reflect"
)

// Controller wraps a List in a channel-based interface.
type Controller struct {
	// state is the internal state managed by the Controller.
	state Controllable

	// clients is the set of Controller-facing client channel pairs.
	// Each client that subscribes gets a Client struct with the other sides.
	// Each client maps to its current index in cselects.
	clients map[coclient]int

	// cselects is the list of cases, one per client, used in the connector select loop.
	// It gets rebuilt every time a client connects or disconnects.
	cselects []reflect.SelectCase

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

	// Done is the channel through which the Controller tells transmitters
	// that the client has shut down.
	// It does so by dropping Done.
	Done <-chan struct{}
}

// Send tries to send a request on a Client.
// It returns false if the Client has shut down.
//
// Send is just sugar over a Select between Tx and Done, and it is
// ok to do this manually using the channels themselves.
func (c *Client) Send(r Request) bool {
	select {
	case c.Tx <- r:
	case <-c.Done:
		return false
	}
	return true
}

// Copy copies a Client, creating a new handle to the Client's Controller.
// The new Client will be separate from this Client: it is ok to dispose of the
// original.
//
// Under the hood, this causes a request to be sent to the Controller goroutine,
// so the Copy will only succeed when the Controller is able to process it.
//
// If Copy returns an error, then the Controller shut down during the copy.
func (c *Client) Copy() (*Client, error) {
	reply := make(chan Response)
	if !c.Send(Request{
		Origin: RequestOrigin{
			Tag:     "",
			ReplyTx: reply,
		},
		Body: newClientRequest{},
	}) {
		return nil, fmt.Errorf("controller shut down while copying")
	}
	var ncli *Client
	for {
		// TODO(@MattWindsor91): be more robust if these don't appear
		// in order
		r := <-reply
		switch b := r.Body.(type) {
		case newClientResponse:
			ncli = b.Client
		case AckResponse:
			return ncli, nil
		}
	}
}

// Shutdown asks a Client to shut down its Controller.
// This is equivalent to sending a ShutdownRequest through the Client,
// but handles the various bits of paperwork.
func (c *Client) Shutdown() {
	if c.Tx == nil {
		panic("double shutdown of client")
	}

	reply := make(chan Response)
	if c.Send(Request{
		Origin: RequestOrigin{
			// It doesn't matter what we put here:
			// the only thing that'll contain it is the ACK,
			// which we bin.
			Tag:     "",
			ReplyTx: reply,
		},
		Body: shutdownRequest{},
	}) {
		// Drain the shutdown acknowledgement.
		<-reply
	}
}

// coclient is the type of internal client handles.
type coclient struct {
	// tx is the status update send channel.
	tx chan<- Response

	// rx is the request receiver channel.
	rx <-chan Request

	// done is the shutdown canary channel.
	done chan<- struct{}
}

// makeClient creates a new client and coclient pair.
func makeClient() (Client, coclient) {
	rq := make(chan Request)
	rs := make(chan Response)
	dn := make(chan struct{})
	ccl := coclient{tx: rs, rx: rq, done: dn}
	cli := Client{Tx: rq, Rx: rs, Done: dn}
	return cli, ccl
}

// makeAndAddClient creates a new client and coclient pair, and adds the coclient to c's clients.
func (c *Controller) makeAndAddClient() *Client {
	client, co := makeClient()
	c.clients[co] = -1

	c.rebuildClientSelects()

	return &client
}

// rebuildClientSelects repopulates the list of client select cases.
// It should be run whenever a client connects or disconnects.
func (c *Controller) rebuildClientSelects() {
	c.cselects = make([]reflect.SelectCase, len(c.clients))
	i := 0
	for cl := range c.clients {
		c.cselects[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cl.rx)}
		c.clients[cl] = i
		i++
	}
}

// NewController constructs a new Controller for a given Controllable.
func NewController(c Controllable) (*Controller, *Client) {
	controller := &Controller{
		state:   c,
		clients: make(map[coclient]int),
	}
	client := controller.makeAndAddClient()
	return controller, client
}

// Run runs this Controller's event loop.
func (c *Controller) Run() {
	c.running = true
	for c.running {
		i, value, open := reflect.Select(c.cselects)
		if open {
			// TODO(@MattWindsor91): properly handle if this isn't a Request
			rq, ok := value.Interface().(Request)
			if !ok {
				panic("FIXME: got bad request")
			}

			c.handleRequest(rq)
		} else {
			c.hangUpClientWithCase(i)
		}
	}

	c.hangUpClients()
}

// Close does the disconnection part of a client hangup.
func (c *coclient) Close() {
	close(c.tx)
	close(c.done)
}

// hangUpClients hangs up every connected client.
func (c *Controller) hangUpClients() {
	for cl := range c.clients {
		cl.Close()
	}
	c.clients = make(map[coclient]int)
	c.rebuildClientSelects()
}

// hangUpClientWithCase hangs up the client whose select case is at index i.
func (c *Controller) hangUpClientWithCase(i int) {
	for cl, j := range c.clients {
		if i == j {
			c.hangUpClient(cl)
			return
		}
	}
}

// hangUpClient closes a client's channels and removes it from the client list.
func (c *Controller) hangUpClient(cl coclient) {
	cl.Close()
	delete(c.clients, cl)
	c.rebuildClientSelects()

	// We need at least one client for the Controller to function
	if len(c.clients) == 0 {
		c.running = false
	}
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
	case newClientRequest:
		err = c.handleNewClientRequest(o, body)
	case shutdownRequest:
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

// handleDumpRequest handles a dump with origin o and body b.
func (c *Controller) handleDumpRequest(o RequestOrigin, b DumpRequest) error {
	dumpCb := func(rbody interface{}) {
		c.reply(o, rbody)
	}
	c.state.Dump(dumpCb)

	// Dump requests never fail
	return nil
}

// handleNewClientRequest handles a new client request with origin o and body b.
func (c *Controller) handleNewClientRequest(o RequestOrigin, b newClientRequest) error {
	cl := c.makeAndAddClient()
	c.reply(o, newClientResponse{Client: cl})

	// New client requests never fail
	return nil
}

// handleRoleRequest handles a role request with origin o and body b.
func (c *Controller) handleRoleRequest(o RequestOrigin, b RoleRequest) error {
	c.reply(o, RoleResponse{Role: c.state.RoleName()})

	// Role requests never fail
	return nil
}

// handleShutdownRequest handles a shutdown request with origin o and body b.
func (c *Controller) handleShutdownRequest(o RequestOrigin, b shutdownRequest) error {
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
