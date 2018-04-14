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

// BifrostParser is the interface of types containing controller-specific parser
// and emitter functionality.
// Each Controller creates one, and a Bifrost uses it to translate
// messages for the Controller's Client into Bifrost messages.
type BifrostParser interface {
	ParseBifrostRequest(word string, args []string) (interface{}, error)
	EmitBifrostResponse(tag string, resp interface{}, out chan<- bifrost.Message) error
}

// UnknownWord returns an error for when a Bifrost parser doesn't understand the
// word w.
func UnknownWord(w string) error {
	return fmt.Errorf("unknown word: %s", w)
}

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

	// parser is some type that provides parsers and emitters for Bifrost
	// messages.
	parser BifrostParser

	// reply is the channel this adapter uses to service replies to requests it sends to the client.
	reply chan Response
}

// BifrostClient is a struct containing channels used to talk to a
// Bifrost adapter.
type BifrostClient struct {
	// Tx is the channel for transmitting requests.
	Tx chan<- bifrost.Message

	// Rx is the channel for receiving responses.
	Rx <-chan bifrost.Message

	// Done is a channel that is closed when the Bifrost adapter's
	// upstream has shut down.
	Done <-chan struct{}
}

// Send tries to send a request on a BifrostClient.
// It returns false if the BifrostClient's upstream has shut down.
//
// Send is just sugar over a Select between Tx and Done, and it is
// ok to do this manually using the channels themselves.
func (c *BifrostClient) Send(r bifrost.Message) bool {
	// See Client.Send in controller.go.
	select {
	case c.Tx <- r:
	case <-c.Done:
		return false
	}
	return true
}

// NewBifrost wraps client inside a Bifrost adapter with parsing and emitting
// done by parser.
// It returns a BifrostClient for talking to the adapter.
func NewBifrost(client *Client, parser BifrostParser) (*Bifrost, *BifrostClient) {
	response := make(chan bifrost.Message)
	request := make(chan bifrost.Message)
	reply := make(chan Response)
	done := make(chan struct{})

	bifrost := Bifrost{
		client:   client,
		resMsgTx: response,
		reqMsgRx: request,
		doneTx:   done,
		reply:    reply,
		parser:   parser,
	}

	bcl := BifrostClient{
		Tx:   request,
		Rx:   response,
		Done: done,
	}

	return &bifrost, &bcl
}

func (b *Bifrost) Close() {
	// Don't shut down any of the client's own channels: other code
	// might still try to use them.
	// Don't shut down the controller: it might have more clients.
	close(b.resMsgTx)
	close(b.doneTx)
}

// Run runs the main body of the Bifrost adapter.
// It will immediately send the new client responses to the response channel.
func (b *Bifrost) Run() {
	defer b.Close()

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
			// No need to check b.client.Done:
			// if the controller shuts down, it pull both this
			// channel and Done at the same time.
			if !ok {
				return
			}
			b.handleResponse(rs)
		}
	}
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
	rbody, err := b.bodyFromMessage(m)
	if err != nil {
		return nil, err
	}

	return makeRequest(rbody, m.Tag(), b.reply), nil
}

// bodyFromMessage tries to parse a message as the body of a controller request.
func (b *Bifrost) bodyFromMessage(m bifrost.Message) (interface{}, error) {
	// Standard requests first.
	switch m.Word() {
	case "dump":
		return parseDumpMessage(m.Args())
	default:
		return b.parser.ParseBifrostRequest(m.Word(), m.Args())
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
	tag := bifrostTagOf(rs)

	var err error

	switch r := rs.Body.(type) {
	case AckResponse:
		err = b.handleAck(tag, r)
	case RoleResponse:
		err = b.handleRole(tag, r)
	default:
		err = b.parser.EmitBifrostResponse(tag, r, b.resMsgTx)
	}

	if err != nil {
		b.resMsgTx <- *errorToMessage(tag, err)
	}
}

// bifrostTagOf works out the Bifrost message tag of response rs.
// This is either the broadcast tag, if rs is a broadcast, or the given tag.
func bifrostTagOf(rs Response) string {
	if rs.Broadcast {
		return bifrost.TagBcast
	}
	if rs.Origin == nil {
		panic("non-broadcast response with nil origin")
	}
	return rs.Origin.Tag
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
