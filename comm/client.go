package comm

// This file defines Client, a struct of channels representing a connection to a
// Controller, and related internal types.

import (
	"context"
	"errors"
	"fmt"
	"github.com/UniversityRadioYork/baps3d/bifrost"
)

var (
	// ErrControllerShutDown is the error sent when a Client operation that
	// needs a running Controller tries to run on a Client whose Controller has
	// shut down.
	ErrControllerShutDown = errors.New("this client's controller has shut down")
)

// Client is the type of external Controller client handles.
type Client struct {
	// Tx is the channel through which the Client can send requests to the Controller.
	Tx chan<- Request

	// Rx is the channel on which the Controller sends status update messages.
	Rx <-chan Response
}

// Send tries to send a request on a Client.
// It returns false if the given context has shut down.
//
// Send is just sugar over a Select between Tx and ctx.Done(), and it is
// ok to do this manually using the channels themselves.
func (c *Client) Send(ctx context.Context, r Request) bool {
	done := ctx.Done()
	select {
	case c.Tx <- r:
	case <-done:
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
func (c *Client) Copy(ctx context.Context) (*Client, error) {
	var ncli *Client

	cb := func(r Response) error {
		b, ok := r.Body.(newClientResponse)
		if !ok {
			return fmt.Errorf("got an unexpected response")
		}
		if ncli != nil {
			return fmt.Errorf("got a duplicate client response")
		}
		if b.Client == nil {
			return fmt.Errorf("got a nil client response")
		}

		ncli = b.Client
		return nil
	}

	alive, err := c.SendAndProcessReplies(ctx, "", newClientRequest{}, cb)
	if !alive {
		return nil, ErrControllerShutDown
	}
	if err != nil {
		return nil, err
	}
	if ncli == nil {
		return nil, fmt.Errorf("didn't get a new client")
	}

	return ncli, nil
}

// Shutdown asks a Client to shut down its Controller.
// This is equivalent to sending a ShutdownRequest through the Client,
// but handles the various bits of paperwork.
func (c *Client) Shutdown(ctx context.Context) error {
	cb := func(Response) error {
		return fmt.Errorf("got an unexpected response")
	}
	// We don't care if the controller has already shut down.
	// Client.Shutdown() should be idempotent.
	_, err := c.SendAndProcessReplies(ctx, "", shutdownRequest{}, cb)
	return err
}

// Bifrost tries to get a Bifrost adapter for Client c's Controller.
// This fails if the Controller's state can't understand Bifrost messages.
func (c *Client) Bifrost(ctx context.Context) (*Bifrost, *bifrost.Client, error) {
	var (
		bf  *Bifrost
		bfc *bifrost.Client
	)

	bfset := false

	cb := func(r Response) error {
		b, ok := r.Body.(bifrostParserResponse)
		if !ok {
			return fmt.Errorf("got an unexpected response")
		}
		if bfset {
			return fmt.Errorf("got a duplicate parser response")
		}

		bf, bfc = NewBifrost(c, b)
		bfset = true
		return nil
	}

	alive, err := c.SendAndProcessReplies(ctx, "", bifrostParserRequest{}, cb)
	if !alive {
		return nil, nil, ErrControllerShutDown
	}
	if err != nil {
		return nil, nil, err
	}
	if !bfset {
		return nil, nil, fmt.Errorf("didn't get a parser response")
	}

	return bf, bfc, nil
}

// ProcessRepliesUntilAck drains the channel reply until an AckResponse is
// returned or the channel closes.
// It feeds any non-Ack response bodies received from reply into cb until and
// unless cb returns an error.
//
// It returns the first of these errors to arrive:
// 1) an error if the channel closed before Ack arrived;
// 2) the first error returned by cb;
// 3) any error coming from the AckResponse.
func ProcessRepliesUntilAck(reply <-chan Response, cb func(Response) error) error {
	var cberr error

	for r := range reply {
		if ack, isAck := r.Body.(AckResponse); isAck {
			if cberr != nil {
				return cberr
			}
			return ack.Err
		}

		if cberr == nil {
			cberr = cb(r)
		}
	}
	return fmt.Errorf("reply channel closed before ack received")
}

// SendAndProcessReplies sends a request with tag tag and body body.
// It then uses cb to process any non-Ack replies.
// It returns whether the Client was able to process the message, and any error.
func (c *Client) SendAndProcessReplies(ctx context.Context, tag string, body interface{}, cb func(Response) error) (bool, error) {
	reply := make(chan Response)

	rq := Request{
		Origin: RequestOrigin{Tag: tag, ReplyTx: reply},
		Body:   body,
	}

	if !c.Send(ctx, rq) {
		return false, nil
	}

	return true, ProcessRepliesUntilAck(reply, cb)
}

// coclient is the type of internal client handles.
type coclient struct {
	// tx is the status update send channel.
	tx chan<- Response

	// rx is the request receiver channel.
	rx <-chan Request
}

// Close does the disconnection part of a client hangup.
func (c *coclient) Close() {
	close(c.tx)
}

// makeClient creates a new client and coclient pair, given a parent context.
func makeClient() (Client, coclient) {
	rq := make(chan Request)
	rs := make(chan Response)
	ccl := coclient{tx: rs, rx: rq}
	cli := Client{Tx: rq, Rx: rs}
	return cli, ccl
}
