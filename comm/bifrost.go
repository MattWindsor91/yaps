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
	// reqConTx is the inward channel to which this adapter sends controller requests.
	reqConTx chan<- Request

	// resConRx is the inward channel from which this adapter receives controller requests.
	resConRx <-chan Response

	// resMsgTx is the outward channel to which this adapter sends response messages.
	resMsgTx chan<- bifrost.Message

	// reqMsgRx is the outward channel from which this adapter receives requests.
	reqMsgRx <-chan bifrost.Message

	// requestMap is the map of known Bifrost message words, and their parsers.
	requestMap map[string]RequestParser

	// responseMsgCb is the callback for handling Bifrost responses.
	responseMsgCb ResponseMsgCb

	// reply is the channel this adapter uses to service replies to requests it sends to the client.
	reply chan Response
}

// NewBifrost wraps client inside a Bifrost adapter with request map rmap and response processor respCb.
// It returns a channel for sending request messages, and one for receiving response messages.
func NewBifrost(client *Client, rmap map[string]RequestParser, respCb ResponseMsgCb) (*Bifrost, chan<- bifrost.Message, <-chan bifrost.Message) {
	response := make(chan bifrost.Message)
	request := make(chan bifrost.Message)
	reply := make(chan Response)

	addStandardRequests(rmap)

	bifrost := Bifrost{
		reqConTx:      client.Tx,
		resConRx:      client.Rx,
		resMsgTx:      response,
		reqMsgRx:      request,
		reply:         reply,
		requestMap:    rmap,
		responseMsgCb: respCb,
	}

	return &bifrost, request, response
}

// Run runs the main body of the Bifrost adapter.
// It will immediately send the new client responses to the response channel.
func (b *Bifrost) Run() {
	b.handleNewClientResponses()
MainLoop:
	for {
		select {
		case rq, ok := <-b.reqMsgRx:
			// Closing the message channel is how the client tells us it has disconnected
			if !ok {
				break MainLoop
			}
			b.handleRequest(rq)
		case rs := <-b.reply:
			b.handleResponse(rs)
		case rs, ok := <-b.resConRx:
			// Closing the response channel is how the controller tells us it has shutdown
			if !ok {
				break MainLoop
			}
			b.handleResponse(rs)
		}
	}

	// Don't shut down the controller: it might have more clients.
	close(b.reqConTx)
	close(b.resMsgTx)
}

//
// Request parsing
//

// handleRequest handles the request message rq.
func (b *Bifrost) handleRequest(rq bifrost.Message) {
	request, err := b.fromMessage(rq)
	if err != nil {
		b.resMsgTx <- *errorToMessage(rq.Tag(), err)
		return
	}

	b.reqConTx <- *request
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
func (b *Bifrost) handleNewClientResponses() {
	// SPEC: see http://universityradioyork.github.io/baps3-spec/protocol/core/commands.html

	// OHAI is a Bifrost-ism, so we don't bother asking the Client about it
	b.resMsgTx <- *bifrost.NewMessage(bifrost.TagBcast, bifrost.RsOhai).AddArg(pversion).AddArg(sversion)

	// We don't use b.reply here, because we want to suppress ACK.
	ncreply := make(chan Response)
	b.reqConTx <- *makeRequest(RoleRequest{}, bifrost.TagBcast, ncreply)
	b.handleResponsesUntilAck(ncreply)
	b.reqConTx <- *makeRequest(DumpRequest{}, bifrost.TagBcast, ncreply)
	b.handleResponsesUntilAck(ncreply)
}

// handleResponsesUntilAck handles responses on channel c until it receives ACK or the channel closes.
func (b *Bifrost) handleResponsesUntilAck(c <-chan Response) {
	for r := range c {
		if _, isAck := r.Body.(AckResponse); isAck {
			return
		}

		b.handleResponse(r)
	}
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
