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

	// msgs is the Bifrost messages channel for this adapter.
	msgs chan bifrost.Message
}

// NewBifrost wraps client inside a Bifrost adapter.
// It returns a bidirectional message channel that can be used to talk to it.
func NewBifrost(client *Client) (*Bifrost, chan bifrost.Message) {
	msgs := make(chan bifrost.Message)
	return &Bifrost{ client: client, msgs: msgs }, msgs
}

// Run runs the main body of the Bifrost adapter.
func (b *Bifrost) Run() {
	for {
		select {
		case rq := <-b.msgs:
			request, err := parseMessage(rq)
			if err != nil {
				b.msgs <- *errorToMessage(rq.Tag(), err)
			} else {
				b.client.Tx <- *request
			}
		case rs := <-b.client.Rx:
			fmt.Println("TODO: got broadcast", rs)
		}
	}
}

// parseMessage tries to parse a message as a controller request.
func parseMessage(m bifrost.Message) (*Request, error) {
	requester, err := parseMessageTail(m.Word(), m.Args())
	if err != nil {
		return nil, err
	}

	return &Request {
		Origin: RequestOrigin { Tag: m.Tag() },
		Body: requester,
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
		return SetAutoModeRequest{ AutoMode: amode }, nil
	default:
		return nil, fmt.Errorf("unknown word: %s", word)
	}
}

// errorToMessage converts the error e to a Bifrost message sent to tag t.
func errorToMessage(t string, e error) *bifrost.Message {
	// TODO(@MattWindsor91): figure out whether e is a WHAT or a FAIL.
	return bifrost.NewMessage(t, "WHAT").AddArg(e.Error())
}
