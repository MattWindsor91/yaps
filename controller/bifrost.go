package controller

// File comm/bifrost.go provides types and functions for creating bridges between Controllers and the Bifrost protocol.

import (
	"context"
	"fmt"

	"github.com/UniversityRadioYork/bifrost-go/core"

	"github.com/UniversityRadioYork/bifrost-go/comm"
	"github.com/UniversityRadioYork/bifrost-go/message"
)

// sversion is the Baps3D semantic server version.
const sversion = "yaps-0.0.0"

// UnknownWord returns an error for when a Bifrost parser doesn't understand the
// word w.
func UnknownWord(w string) error {
	return fmt.Errorf("unknown word: %s", w)
}

// Bifrost is the type of adapters from Controller clients to Bifrost.
type Bifrost struct {
	// Client is the inward client the Bifrost adapter is using to talk to
	// the Controller.
	client *Client

	// bifrost is the endpoint being used to talk to a Bifrost client.
	bifrost *comm.Endpoint

	// reply is the channel this adapter uses to service replies to requests it sends to the client.
	reply chan Response
}

// NewBifrost wraps client inside a Bifrost adapter with parsing and emitting
// done by parser.
// It returns a bifrost.Endpoint for talking to the adapter.
func NewBifrost(client *Client) (*Bifrost, *comm.Endpoint) {
	reply := make(chan Response)

	pubEnd, privEnd := comm.NewEndpointPair()

	bif := Bifrost{
		client:  client,
		bifrost: privEnd,
		reply:   reply,
	}

	return &bif, pubEnd
}

func (b *Bifrost) respond(m message.Message) {
	b.bifrost.Tx <- m
}

func (b *Bifrost) close() {
	close(b.bifrost.Tx)
}

// Run runs the main body of the Bifrost adapter.
// It will immediately send the new client responses to the response channel.
func (b *Bifrost) Run(ctx context.Context) {
	defer b.close()

	if !b.handleNewClientResponses(ctx) {
		return
	}

	for {
		// Closing the message channel is how the client tells us it has disconnected.
		// Closing the response channel, or refusing a message,
		// tells us the controller has shut down.
		// Either way, we need to close.

		select {
		case rq, ok := <-b.bifrost.Rx:
			if !ok || !b.handleRequest(ctx, rq) {
				return
			}
		case rs := <-b.reply:
			b.handleResponseForwardingError(rs)
		case rs, ok := <-b.client.Rx:
			// No need to check b.client.Done:
			// if the controller shuts down, it pull both this
			// channel and Done at the same time.
			if !ok {
				return
			}
			b.handleResponseForwardingError(rs)
		}
	}
}

//
// Request parsing
//

// handleRequest handles the request message rq.
// It returns whether the client is still able to handle
// requests.
func (b *Bifrost) handleRequest(ctx context.Context, rq message.Message) bool {
	request, err := b.fromMessage(rq)
	if err != nil {
		b.respond(*errorToMessage(rq.Tag(), err))
		return true
	}

	return b.client.Send(ctx, *request)
}

// fromMessage tries to parse a message as a controller request.
func (b *Bifrost) fromMessage(m message.Message) (*Request, error) {
	rbody, err := b.bodyFromMessage(m)
	if err != nil {
		return nil, err
	}

	return makeRequest(rbody, m.Tag(), b.reply), nil
}

// bodyFromMessage tries to parse a message as the body of a controller request.
func (b *Bifrost) bodyFromMessage(m message.Message) (interface{}, error) {
	// Standard requests first.
	switch m.Word() {
	case "dump":
		return parseDumpMessage(m.Args())
	default:
		return comm.ParseMessage(&m)
	}
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
// It returns true if the client context hasn't hung up midway through.
func (b *Bifrost) handleNewClientResponses(ctx context.Context) bool {
	// SPEC: see http://universityradioyork.github.io/baps3-spec/protocol/core/commands.html

	// OHAI is a Bifrost-ism, so we don't bother asking the Client about it
	b.sendOhai()

	// We don't use b.reply here, because we want to suppress ACK.
	ncreply := make(chan Response)
	if !b.client.Send(ctx, *makeRequest(RoleRequest{}, message.TagBcast, ncreply)) {
		return false
	}
	if ProcessRepliesUntilAck(ncreply, b.handleResponse) != nil {
		return false
	}
	if !b.client.Send(ctx, *makeRequest(DumpRequest{}, message.TagBcast, ncreply)) {
		return false
	}
	return ProcessRepliesUntilAck(ncreply, b.handleResponse) == nil
}

func (b *Bifrost) sendOhai() {
	ohai := core.OhaiResponse{
		ProtocolVer: core.ThisProtocolVer,
		ServerVer:   sversion,
	}
	b.respond(*ohai.Message(message.TagBcast))
}

// handleResponseForwardingError handles a controller response rs, forwarding
// the error as a // message.
func (b *Bifrost) handleResponseForwardingError(rs Response) {
	if err := b.handleResponse(rs); err != nil {
		b.respond(*errorToMessage(bifrostTagOf(rs), err))
	}
}

// handleResponse handles a controller response rs.
func (b *Bifrost) handleResponse(rs Response) error {
	tag := bifrostTagOf(rs)

	switch r := rs.Body.(type) {
	case DoneResponse:
		return b.handleAck(tag, r)
	case core.IamaResponse:
		return b.handleRole(tag, r)
	case comm.Messager:
		b.bifrost.Send(context.Background(), *r.Message(tag))
		return nil
	default:
		return fmt.Errorf("can't turn %v into a message", r)
	}
}

// bifrostTagOf works out the Bifrost message tag of response rs.
// This is either the broadcast tag, if rs is a broadcast, or the given tag.
func bifrostTagOf(rs Response) string {
	if rs.Broadcast {
		return message.TagBcast
	}
	if rs.Origin == nil {
		panic("non-broadcast response with nil origin")
	}
	return rs.Origin.Tag
}

// handleAck handles converting an DoneResponse r into messages for tag t.
// If the ACK had an error, it is propagated down.
func (b *Bifrost) handleAck(t string, r DoneResponse) error {

	if r.Err != nil {
		return r.Err
	}

	b.respond(*message.New(t, core.RsAck).AddArgs("OK", "success"))
	return nil
}

// handleRole handles converting a IamaResponse r into messages for tag t.
func (b *Bifrost) handleRole(t string, r core.IamaResponse) error {
	b.respond(*((&r).Message(t)))
	return nil
}

// errorToMessage converts the error e to a Bifrost message sent to tag t.
func errorToMessage(t string, e error) *message.Message {
	// TODO(@MattWindsor91): figure out whether e is a WHAT or a FAIL.
	return message.New(t, core.RsAck).AddArgs("WHAT", e.Error())
}
