package list

// This file contains a bridge between the Bifrost list protocol and the
// list Controller requests and responses.

import (
	"fmt"
	"strconv"

	"github.com/UniversityRadioYork/baps3d/bifrost"
)

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
	requestMap map[string]requestParser

	// reply is the channel this adapter uses to service replies to requests it sends to the client.
	reply chan Response
}

// NewBifrost wraps client inside a Bifrost adapter.
// It returns a channel for sending request messages, and one for receiving response messages.
func NewBifrost(client *Client) (*Bifrost, chan<- bifrost.Message, <-chan bifrost.Message) {
	response := make(chan bifrost.Message)
	request := make(chan bifrost.Message)
	reply := make(chan Response)

	// TODO(@MattWindsor91): when generalising, make the tables get passed in.

	bifrost := Bifrost{
		reqConTx:   client.Tx,
		resConRx:   client.Rx,
		resMsgTx:   response,
		reqMsgRx:   request,
		reply:      reply,
		requestMap: newRequestMap(),
	}

	return &bifrost, request, response
}

// Run runs the main body of the Bifrost adapter.
// It will immediately send the new client responses to the response channel.
func (b *Bifrost) Run() {
	b.handleNewClientResponses()
	for {
		select {
		case rq := <-b.reqMsgRx:
			b.handleRequest(rq)
		case rs := <-b.reply:
			b.handleResponse(rs)
		case rs := <-b.resConRx:
			b.handleResponse(rs)
		}
	}
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

// requestParser is the type of request parsers.
type requestParser func([]string) (interface{}, error)

// newRequestMap builds the request parser map.
func newRequestMap() map[string]requestParser {
	return map[string]requestParser{
		"dump": parseDumpMessage,
		"auto": parseAutoMessage,
	}
}

// parseDumpMessage tries to parse a 'dump' message.
func parseDumpMessage(args []string) (interface{}, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("bad arity")
	}

	return DumpRequest{}, nil
}

// parseAutoMessage tries to parse an 'auto' message.
func parseAutoMessage(args []string) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("bad arity")
	}

	amode, err := ParseAutoMode(args[0])
	if err != nil {
		return nil, err
	}

	return SetAutoModeRequest{AutoMode: amode}, nil
}

//
// Response emitting
//

// handleNewClientResponses handles the new client responses (OHAI, IAMA, etc).
func (b *Bifrost) handleNewClientResponses() {
	// We don't use b.reply here, because we want to suppress ACK.
	ncreply := make(chan Response)

	// TODO(@MattWindsor91): OHAI
	// TODO(@MattWindsor91): IAMA
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
	case AutoModeResponse:
		err = b.handleAutoMode(tag, r)
	case FreezeResponse:
		err = b.handleFreeze(tag, r)
	case ItemResponse:
		err = b.handleItem(tag, r)
	default:
		err = fmt.Errorf("response with no message equivalent: %v", r)
	}

	if err != nil {
		// TODO(@MattWindsor91): propagate?
		fmt.Println("response error:", err.Error())
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

// handleAutoMode handles converting an AutoModeResponse r into messages for tag t.
func (b *Bifrost) handleAutoMode(t string, r AutoModeResponse) error {
	b.resMsgTx <- *bifrost.NewMessage(t, "AUTO").AddArg(r.AutoMode.String())
	return nil
}

// handleFreeze handles converting an FreezeResponse r into messages for tag t.
func (b *Bifrost) handleFreeze(t string, r FreezeResponse) error {
	b.resMsgTx <- *bifrost.NewMessage(t, "COUNTL").AddArg(strconv.Itoa(len(r)))

	// The next bit is the same as if we were loading the items--
	// so we reuse the logic.
	for i, item := range r {
		ilr := ItemResponse{
			Index: i,
			Item:  item,
		}

		if err := b.handleItem(t, ilr); err != nil {
			return err
		}
	}

	return nil
}

// handleItem handles converting an ItemResponse r into messages for tag t.
func (b *Bifrost) handleItem(t string, r ItemResponse) error {
	var word string
	switch r.Item.Type() {
	case ItemTrack:
		word = "floadl"
	case ItemText:
		word = "tloadl"
	default:
		return fmt.Errorf("unknown item type %v", r.Item.Type())
	}

	b.resMsgTx <- *bifrost.NewMessage(t, word).AddArg(strconv.Itoa(r.Index)).AddArg(r.Item.Hash()).AddArg(r.Item.Payload())
	return nil
}

// errorToMessage converts the error e to a Bifrost message sent to tag t.
func errorToMessage(t string, e error) *bifrost.Message {
	// TODO(@MattWindsor91): figure out whether e is a WHAT or a FAIL.
	return bifrost.NewMessage(t, bifrost.RsAck).AddArg("WHAT").AddArg(e.Error())
}
