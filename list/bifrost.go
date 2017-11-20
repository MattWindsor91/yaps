package list

// File list/bifrost.go contains List-specific Bifrost marshalling logic.
// - See `comm/bifrost.go` for the common marshalling logic.

import (
	"fmt"
	"strconv"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/UniversityRadioYork/baps3d/comm"
)

// NewBifrost wraps client inside a Bifrost adapter for lists.
func NewBifrost(client *comm.Client) (*comm.Bifrost, chan<- bifrost.Message, <-chan bifrost.Message) {
	return comm.NewBifrost(
		client,
		map[string]comm.RequestParser{
			"auto": parseAutoMessage,
			"sel":  parseSelMessage,
		},
		handleResponse,
	)
}

//
// Request parsers
//

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

/// parseSelMEssage tries to parse a 'sel' message.
func parseSelMessage(args []string) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("bad arity")
	}

	index, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, err
	}
	hash := args[1]

	return SetSelectRequest{Index: index, Hash: hash}, nil
}

//
// Response emitting
//

// handleResponse handles a controller response with tag tag and body rbody.
// It sends response messages to msgTx.
func handleResponse(tag string, rbody interface{}, msgTx chan<- bifrost.Message) (err error) {
	switch r := rbody.(type) {
	case AutoModeResponse:
		err = handleAutoMode(tag, r, msgTx)
	case FreezeResponse:
		err = handleFreeze(tag, r, msgTx)
	case ItemResponse:
		err = handleItem(tag, r, msgTx)
	case SelectResponse:
		err = handleSelect(tag, r, msgTx)
	default:
		err = fmt.Errorf("response with no message equivalent: %v", r)
	}

	return
}

// handleAutoMode handles converting an AutoModeResponse r into messages for tag t.
func handleAutoMode(t string, r AutoModeResponse, msgTx chan<- bifrost.Message) error {
	msgTx <- *bifrost.NewMessage(t, "AUTO").AddArg(r.AutoMode.String())
	return nil
}

// handleFreeze handles converting a FreezeResponse r into messages for tag t.
func handleFreeze(t string, r FreezeResponse, msgTx chan<- bifrost.Message) error {
	msgTx <- *bifrost.NewMessage(t, "COUNTL").AddArg(strconv.Itoa(len(r)))

	// The next bit is the same as if we were loading the items--
	// so we reuse the logic.
	for i, item := range r {
		ilr := ItemResponse{
			Index: i,
			Item:  item,
		}

		if err := handleItem(t, ilr, msgTx); err != nil {
			return err
		}
	}

	return nil
}

// handleItem handles converting an ItemResponse r into messages for tag t.
func handleItem(t string, r ItemResponse, msgTx chan<- bifrost.Message) error {
	var word string
	switch r.Item.Type() {
	case ItemTrack:
		word = "FLOADL"
	case ItemText:
		word = "TLOADL"
	default:
		return fmt.Errorf("unknown item type %v", r.Item.Type())
	}

	msgTx <- *bifrost.NewMessage(t, word).AddArg(strconv.Itoa(r.Index)).AddArg(r.Item.Hash()).AddArg(r.Item.Payload())
	return nil
}

// handleSelect handles converting a SelectResponse r into messages for tag t.
func handleSelect(t string, r SelectResponse, msgTx chan<- bifrost.Message) error {
	msg := *bifrost.NewMessage(t, "SEL").AddArg(strconv.Itoa(r.Index)).AddArg(r.Hash)
	msgTx <- msg
	return nil
}
