package list

// This file contains a bridge between the Bifrost list protocol and the
// list Controller requests and responses.

import (
	"fmt"

	"github.com/UniversityRadioYork/baps3d/bifrost"
)

// Bifrost is the type of adapters from list Controller clients to Bifrost.
type Bifrost struct {
	// client is the list Controller client.
	client *Client

	// responseTx is the channel to which this adapter sends responses.
	responseTx chan<- bifrost.Message

	// requestRx is the channel to which this adapter sends requests.
	requestRx <-chan bifrost.Message
}

// NewBifrost wraps client inside a Bifrost adapter.
// It returns a channel for sending request messages, and one for receiving response messages.
func NewBifrost(client *Client) (*Bifrost, chan<- bifrost.Message, <-chan bifrost.Message) {
	response := make(chan bifrost.Message)
	request := make(chan bifrost.Message)
	return &Bifrost{client: client, responseTx: response, requestRx: request}, request, response
}

// Run runs the main body of the Bifrost adapter.
func (b *Bifrost) Run() {
	for {
		select {
		case rq := <-b.requestRx:
			request, err := fromMessage(rq)
			if err != nil {
				b.responseTx <- *errorToMessage(rq.Tag(), err)
			} else {
				b.client.Tx <- *request
			}
		case rs := <-b.client.Rx:
			response, err := toMessage(rs)
			if err != nil {
				fmt.Println("internal message emit error:", err)
			} else {
				b.responseTx <- *response
			}
		}
	}
}

// fromMessage tries to parse a message as a controller request.
func fromMessage(m bifrost.Message) (*Request, error) {
	requester, err := parseMessageTail(m.Word(), m.Args())
	if err != nil {
		return nil, err
	}

	return &Request{
		Origin: RequestOrigin{Tag: m.Tag()},
		Body:   requester,
	}, nil
}

// parseMessageTail tries to parse the word and arguments of a message as a controller request payload.
func parseMessageTail(word string, args []string) (interface{}, error) {
	switch word {
	case "auto":
		if len(args) != 1 {
			return nil, fmt.Errorf("bad arity")
		}
		amode, err := ParseAutoMode(args[0])
		if err != nil {
			return nil, err
		}
		return SetAutoModeRequest{AutoMode: amode}, nil
	default:
		return nil, fmt.Errorf("unknown word: %s", word)
	}
}

// toMessage tries to convert a response rs into a Bifrost message sent to tag t..
func toMessage(rs Response) (*bifrost.Message, error) {
	tag := rs.Tag()
	
	switch r := rs.Body.(type) {
	case AutoModeResponse:
		return bifrost.NewMessage(tag, "AUTO").AddArg(r.AutoMode.String()), nil
	default:
		return nil, fmt.Errorf("response with no message equivalent: %A", r)
	}
}

// errorToMessage converts the error e to a Bifrost message sent to tag t.
func errorToMessage(t string, e error) *bifrost.Message {
	// TODO(@MattWindsor91): figure out whether e is a WHAT or a FAIL.
	return bifrost.NewMessage(t, "WHAT").AddArg(e.Error())
}
