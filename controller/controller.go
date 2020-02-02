package controller

// This file defines Controller, an object that encapsulates part of a baps3d server's state and provides a Request/Response interface to it.
// The baps3d state must satisfy the 'Controllable' interface.

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/UniversityRadioYork/baps3d/bifrost/corecmd"

	"github.com/UniversityRadioYork/baps3d/bifrost"
)

var (
	// ErrControllerCannotSpeakBifrost is the error sent when a Client requests
	// a Bifrost adapter for a Controller, but its Controllable state doesn't
	// implement BifrostParser.
	ErrControllerCannotSpeakBifrost = errors.New("this controller's state can't parse Bifrost messages")
)

// Controller wraps a baps3d service in a channel-based interface.
// The service must satisfy the 'Controllable' interface.
type Controller struct {
	// state is the internal state managed by the Controller.
	state Controllable

	// clients is the set of Controller-facing client channel pairs.
	// Each client that subscribes gets a Client struct with the other sides.
	// Each client maps to its current index in cselects.
	clients map[coclient]int

	// mounts is the mapping of mount-point names to Clients that represent 'mounted' Controllers.
	mounts map[string]Client

	// cselects is the list of cases, one per client, used in the connector select loop.
	// It gets rebuilt every time a client connects or disconnects.
	//
	// We use a set of cases rather than a single multiplexed request channel
	// to let clients notify the Controller of disconnection by closing its own
	// channel.
	cselects []reflect.SelectCase

	// running is the internal is-running flag.
	// When this is set to false, the controller loop will exit.
	running bool
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
func (c *Controller) Run(ctx context.Context) {
	c.running = true
	for c.running {
		i, value, open := reflect.Select(c.cselects)
		if open {
			// TODO(@MattWindsor91): properly handle if this isn't a Request
			rq, ok := value.Interface().(Request)
			if !ok {
				panic("FIXME: got bad request")
			}

			c.handleRequest(ctx, rq)
		} else {
			c.hangUpClientWithCase(i)
		}
	}

	c.hangUpClients()
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
func (c *Controller) handleRequest(ctx context.Context, rq Request) {
	var err error

	o := rq.Origin
	switch body := rq.Body.(type) {
	case RoleRequest:
		err = c.handleRoleRequest(o, body)
	case OnRequest:
		err = c.handleOnRequest(ctx, o, body)
	case DumpRequest:
		err = c.handleDumpRequest(o, body)
	case newClientRequest:
		err = c.handleNewClientRequest(o, body)
	case shutdownRequest:
		err = c.handleShutdownRequest(o, body)
	case bifrostParserRequest:
		err = c.handleBifrostParserRequest(o, body)
	default:
		err = c.handleStateSpecificRequest(o, body)
	}

	ack := DoneResponse{err}
	c.reply(o, ack)
}

func (c *Controller) handleStateSpecificRequest(o RequestOrigin, body interface{}) error {
	replyCb := func(rbody interface{}) {
		c.reply(o, rbody)
	}
	return c.state.HandleRequest(replyCb, c.broadcast, body)
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

// handleOnRequest handles an 'on' request with origin o and body b.
func (c *Controller) handleOnRequest(ctx context.Context, o RequestOrigin, b OnRequest) error {
	m, ok := c.mounts[b.MountPoint]
	if !ok {
		return fmt.Errorf("no such mount point: %s", b.MountPoint)
	}
	if !m.Send(ctx, b.Request) {
		return fmt.Errorf("couldn't send to mount point: %s", b.MountPoint)
	}
	return nil
}

// handleRoleRequest handles a role request with origin o and body b.
func (c *Controller) handleRoleRequest(o RequestOrigin, b RoleRequest) error {
	c.reply(o, corecmd.IamaResponse{Role: c.state.RoleName()})

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

// handleBifrostParserRequest handles a get-Bifrost-parser request with origin o and body b.
func (c *Controller) handleBifrostParserRequest(o RequestOrigin, b bifrostParserRequest) error {
	bp, ok := c.state.(bifrost.Parser)
	if !ok {
		return ErrControllerCannotSpeakBifrost
	}

	c.reply(o, bifrostParserResponse(bp))
	return nil
}

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
