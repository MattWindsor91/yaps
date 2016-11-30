package list

// This file defines Controller, a wrapper for List that exposes a channel interface.
// It is used to lift a List into a goroutine communicating with players and clients.
// For the protocol used by the Controller, see 'messages.go'.

import (
	"fmt"
	"reflect"
)


// Controller wraps a List in a channel-based interface.
type Controller struct {
	// list is the internal list managed by the Controller.
	list *List

	// requesters is the set of request channels listened to by the Controller.
	requesters []<-chan Request

	// TODO(CaptainHayashi): broadcast channels.
}

// NewController constructs a new Controller for a given List.
func NewController(l *List) *Controller {
	return &Controller{
		list: l,
		requesters: []<-chan Request{},
	}
}

// Run runs this Controller's event loop.
func (c *Controller) Run() {
	cases := make([]reflect.SelectCase, len(c.requesters))
	for i, ch := range c.requesters {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}

	for {
		_, value, ok := reflect.Select(cases)
		if ok {
			// TODO(CaptainHayashi): process value
			fmt.Println(value)
		}
	}
	// TODO(CaptainHayashi): some way to die
}
