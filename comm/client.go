package comm

// This file defines Client, a struct of channels representing a connection to a
// Controller, and related internal types.

import "fmt"

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

// Bifrost tries to get a Bifrost adapter for Client c's Controller.
// This fails if the Controller's state can't understand Bifrost messages.
func (c *Client) Bifrost() (*Bifrost, *BifrostClient, error) {
	reply := make(chan Response)
	if !c.Send(Request{
		Origin: RequestOrigin{
			Tag:     "",
			ReplyTx: reply,
		},
		Body: bifrostParserRequest{},
	}) {
		return nil, nil, fmt.Errorf("controller shut down while getting a Bifrost")
	}
	var (
		bf       *Bifrost
		bfc      *BifrostClient
		innerErr error
	)

	bfset := false

	cb := func(rb interface{}) {
		if innerErr != nil {
			return
		}

		b, ok := rb.(bifrostParserResponse)
		if !ok {
			innerErr = fmt.Errorf("got an unexpected response")
		}
		if bfset {
			innerErr = fmt.Errorf("got a duplicate parser response")
		}

		bf, bfc = NewBifrost(c, b)
	}
	if aerr := ProcessRepliesUntilAck(reply, cb); aerr != nil {
		return nil, nil, aerr
	}
	if innerErr != nil {
		return nil, nil, innerErr
	}
	if !bfset {
		return nil, nil, fmt.Errorf("didn't get a parser response")
	}

	return bf, bfc, nil
}

// ProcessRepliesUntilAck feeds response bodies from the channel reply into cb
// until an AckResponse is returned or the channel closes.
// It returns any error coming from the AckResponse, or an error if the channel
// closed before one arrived.
func ProcessRepliesUntilAck(reply <-chan Response, cb func(interface{})) error {
	for r := range reply {
		if ack, isAck := r.Body.(AckResponse); isAck {
			return ack.Err
		}

		cb(r.Body)
	}
	return fmt.Errorf("reply channel closed before ack received")
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

// Close does the disconnection part of a client hangup.
func (c *coclient) Close() {
	close(c.tx)
	close(c.done)
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
