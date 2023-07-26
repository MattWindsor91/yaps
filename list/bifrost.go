package list

// File list/bifrost.go implements BifrostParser for List.
// - See `comm/bifrost.go` for the common marshalling logic.

import (
	"fmt"
	"strconv"

	"github.com/UniversityRadioYork/bifrost-go/message"

	"github.com/MattWindsor91/yaps/controller"
)

// ParseBifrostRequest handles Bifrost parsing for List controllers.
func (l *List) ParseBifrostRequest(word string, args []string) (interface{}, error) {
	switch word {
	case "auto":
		return parseAutoMessage(args)
	case "floadl":
		return parseFloadlMessage(args)
	case "sel":
		return parseSelMessage(args)
	case "tloadl":
		return parseTloadlMessage(args)
	default:
		return nil, controller.UnknownWord(word)
	}
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

// parseFloadlMessage tries to parse a 'floadl' message.
func parseFloadlMessage(args []string) (interface{}, error) {
	return parseItemAddMessage(NewTrack, args)
}

// parseSelMessage tries to parse a 'sel' message.
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

// parseTloadlMessage tries to parse a 'tloadl' message.
func parseTloadlMessage(args []string) (interface{}, error) {
	return parseItemAddMessage(NewText, args)
}

// parseItemAddMessage tries to parse a '*loadl' message with arguments args.
// We have already decided which type of item we're adding and stored its constructor in con.
func parseItemAddMessage(con func(string, string) *Item, args []string) (interface{}, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("bad arity")
	}

	index, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, err
	}
	hash := args[1]
	payload := args[2]

	item := con(hash, payload)
	return AddItemRequest{Index: index, Item: *item}, nil
}

//
// Response emitting
//

// EmitBifrostResponse handles a controller response with tag tag and body rbody.
// It sends response messages to msgTx.
func (l *List) EmitBifrostResponse(tag string, rbody interface{}, msgTx chan<- message.Message) (err error) {
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
func handleAutoMode(t string, r AutoModeResponse, msgTx chan<- message.Message) error {
	msgTx <- *message.New(t, "AUTO").AddArgs(r.AutoMode.String())
	return nil
}

// handleFreeze handles converting a FreezeResponse r into messages for tag t.
func handleFreeze(t string, r FreezeResponse, msgTx chan<- message.Message) error {
	msgTx <- *message.New(t, "COUNTL").AddArgs(strconv.Itoa(len(r)))

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
func handleItem(t string, r ItemResponse, msgTx chan<- message.Message) error {
	var word string
	switch r.Item.Type() {
	case ItemTrack:
		word = "FLOADL"
	case ItemText:
		word = "TLOADL"
	default:
		return fmt.Errorf("unknown item type %v", r.Item.Type())
	}

	msgTx <- *message.New(t, word).AddArgs(strconv.Itoa(r.Index), r.Item.Hash(), r.Item.Payload())
	return nil
}

// handleSelect handles converting a SelectResponse r into messages for tag t.
func handleSelect(t string, r SelectResponse, msgTx chan<- message.Message) error {
	msg := *message.New(t, "SEL").AddArgs(strconv.Itoa(r.Index), r.Hash)
	msgTx <- msg
	return nil
}
