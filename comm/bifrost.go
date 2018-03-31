package comm

// File comm/bifrost.go provides types and functions for creating bridges between Controllers and the Bifrost protocol.

import (
	"fmt"

	"github.com/UniversityRadioYork/baps3d/bifrost"
)

// pversion is the Bifrost semantic protocol version.
var pversion = "bifrost-0.0.0"

// sversion is the Baps3D semantic server version.
var sversion = "baps3d-0.0.0"

// RequestParser is the type of request parsing functions.
type RequestParser func([]string) (interface{}, error)

// ResponseMsgCb is the type of response marshalling callbacks.
// It is supplied the response's tag string and body, and a channel for emitting messages.
type ResponseMsgCb func(string, interface{}, chan<- bifrost.Message) error

// Bifrost is the type of adapters from list Controller clients to Bifrost.
type Bifrost struct {
	// Client is the inward client the Bifrost adapter is using to talk to
	// the Controller.
	client *Client
	
	// resMsgTx is the outward channel to which this adapter sends response messages.
	resMsgTx chan<- bifrost.Message

	// reqMsgRx is the outward channel from which this adapter receives requests.
	reqMsgRx <-chan bifrost.Message

	// doneMsgTx is the outward channel on which this adapter
	// sends 'done' signals.
	doneTx chan<- struct{}

	// requestMap is the map of known Bifrost message words, and their parsers.
	requestMap map[string]RequestParser

	// responseMsgCb is the callback for handling Bifrost responses.
	responseMsgCb ResponseMsgCb

	// reply is the channel this adapter uses to service replies to requests it sends to the client.
	reply chan Response
}

// NewBifrost wraps client inside a Bifrost adapter with request map rmap and response processor respCb.
// It returns a channel for sending request messages, and one for receiving response messages.
func NewBifrost(client *Client, rmap map[string]RequestParser, respCb ResponseMsgCb) (*Bifrost, chan<- bifrost.Message, <-chan bifrost.Message, <-chan struct{}) {
	response := make(chan bifrost.Message)
	request := make(chan bifrost.Message)
	reply := make(chan Response)
	done := make(chan struct{})

	// This should be idempotent if we're Fork()ing an existing Bifrost
	addStandardRequests(rmap)

	bifrost := Bifrost{
		client:        client,
		resMsgTx:      response,
		reqMsgRx:      request,
		doneTx:        done,
		reply:         reply,
		requestMap:    rmap,
		responseMsgCb: respCb,
	}

	return &bifrost, request, response, done
}

func (b *Bifrost) hangup() {
	// Don't shut down any of the client's own channels: other code
	// might still try to use them.
	// Don't shut down the controller: it might have more clients.
	close(b.resMsgTx)
	close(b.doneTx)
}

// Run runs the main body of the Bifrost adapter.
// It will immediately send the new client responses to the response channel.
func (b *Bifrost) Run() {
	defer b.hangup()
	if !b.handleNewClientResponses() {
		return
	}

	for {
		// Closing the message channel is how the client tells us it has disconnected.
		// Closing the response channel, or refusing a message,
		// tells us the controller has shut down.
		// Either way, we need to close.
		
		select {
		case rq, ok := <-b.reqMsgRx:
			if !ok || !b.handleRequest(rq) {
				return
			}
		case rs := <-b.reply:
			b.handleResponse(rs)
		case rs, ok := <-b.client.Rx:
			// Closing the response channel is how the controller tells us it has shutdown
			if !ok {
				return
			}
			b.handleResponse(rs)
		}
	}
}

// Fork creates a new Bifrost adapter with the same parsing logic as b.
func (b *Bifrost) Fork(client *Client) (*Bifrost, chan<- bifrost.Message, <-chan bifrost.Message, <-chan struct{}) {
	// TODO(@MattWindsor91): split config from Bifrost, copy config only.
	return NewBifrost(client, b.requestMap, b.responseMsgCb)
}

//
// Request parsing
//

// handleRequest handles the request message rq.
// It returns whether or not the client is still able to handle
// requests.
func (b *Bifrost) handleRequest(rq bifrost.Message) bool {
	request, err := b.fromMessage(rq)
	if err != nil {
		b.resMsgTx <- *errorToMessage(rq.Tag(), err)
		return true
	}

	return b.client.Send(*request)
}

// fromMessage tries to parse a message as a controller request.
func (b *Bifrost) fromMessage(m bifrost.Message) (*Request, error) {
	parser, ok := b.requestMap[m.Word()]
	if !ok {
		return nil, fmt.Errorf("unknown word: %s", m.Word())
	}

	rbody, err := parser(m.Args())
	if err != nil {
		return nil, err
	}

	return makeRequest(rbody, m.Tag(), b.reply), nil
}

// makeRequest creates a request with body rbody, tag tag, and reply channel rch.
// m may be nil.
func makeRequest(rbody interface{}, tag string, rch chan<- Response) *Request {
	origin := RequestOrigin{
		Tag:     tag,
		ReplyTx: rch,
	}
	request := Request{
		Origin: origin,
		Body:   rbody,
	}
	return &request
}

//
// Standard request parsers
//

// addStandardRequests adds the standard request parsers to rmap.
func addStandardRequests(rmap map[string]RequestParser) {
	rmap["dump"] = parseDumpMessage
}

// parseDumpMessage tries to parse a 'dump' message.
func parseDumpMessage(args []string) (interface{}, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("bad arity")
	}

	return DumpRequest{}, nil
}

//
// Response emitting
//

// handleNewClientResponses handles the new client responses (OHAI, IAMA, etc).
// It returns true if the client hasn't hung up midway through.
func (b *Bifrost) handleNewClientResponses() bool {
	// SPEC: see http://universityradioyork.github.io/baps3-spec/protocol/core/commands.html

	// OHAI is a Bifrost-ism, so we don't bother asking the Client about it
	b.resMsgTx <- *bifrost.NewMessage(bifrost.TagBcast, bifrost.RsOhai).AddArg(pversion).AddArg(sversion)

	// We don't use b.reply here, because we want to suppress ACK.
	ncreply := make(chan Response)
	if !b.client.Send(*makeRequest(RoleRequest{}, bifrost.TagBcast, ncreply)) {
		return false
	}
	if !b.handleResponsesUntilAck(ncreply) {
		return false
	}
	if !b.client.Send(*makeRequest(DumpRequest{}, bifrost.TagBcast, ncreply)) {
		return false
	}
	return b.handleResponsesUntilAck(ncreply)
}

// handleResponsesUntilAck handles responses on channel c until it receives ACK or the channel closes.
func (b *Bifrost) handleResponsesUntilAck(c <-chan Response) bool {
	for r := range c {
		if _, isAck := r.Body.(AckResponse); isAck {
			return true
		}

		b.handleResponse(r)
	}
	return false
}

// handleResponse handles a controller response rs.
func (b *Bifrost) handleResponse(rs Response) {
	tag := rs.Origin.Tag

	var err error

	switch r := rs.Body.(type) {
	case AckResponse:
		err = b.handleAck(tag, r)
	case RoleResponse:
		err = b.handleRole(tag, r)
	default:
		err = b.responseMsgCb(tag, r, b.resMsgTx)
	}

	if err != nil {
		b.resMsgTx <- *errorToMessage(tag, err)
	}
}

// handleAck handles converting an AckResponse r into messages for tag t.
// If the ACK had an error, it is propagated down.
func (b *Bifrost) handleAck(t string, r AckResponse) error {
	if r.Err != nil {
		return r.Err
	}

	// SPEC: The wording here is specific.
	// SPEC: See https://universityradioyork.github.io/baps3-spec/protocol/core/commands.html
	b.resMsgTx <- *bifrost.NewMessage(t, bifrost.RsAck).AddArg("OK").AddArg("success")
	return nil
}

// handleRole handles converting a RoleResponse r into messages for tag t.
func (b *Bifrost) handleRole(t string, r RoleResponse) error {
	b.resMsgTx <- *bifrost.NewMessage(t, "IAMA").AddArg(r.Role)
	return nil
}

// errorToMessage converts the error e to a Bifrost message sent to tag t.
func errorToMessage(t string, e error) *bifrost.Message {
	// TODO(@MattWindsor91): figure out whether e is a WHAT or a FAIL.
	return bifrost.NewMessage(t, bifrost.RsAck).AddArg("WHAT").AddArg(e.Error())
}
