package list

// File controller.go defines the specific Controller logic for lists.

import (
	"fmt"

	"github.com/UniversityRadioYork/baps3d/comm"
)

// NewControlledList constructs a new List and wraps it in a Controller.
func NewControlledList() (*comm.Controller, *comm.Client) {
	return comm.NewController(New())
}

// RoleName gives the role name for a List Controller.
func (l *List) RoleName() string {
	return "list"
}

//
// Dump logic
//

// automodeResponse returns l's automode as a response.
func (l *List) autoModeResponse() AutoModeResponse {
	return AutoModeResponse{AutoMode: l.AutoMode()}
}

// selectResponse returns l's selection as a response.
func (l *List) selectResponse() SelectResponse {
	index, item := l.Selection()

	var hash string
	if item == nil {
		if index != -1 {
			panic("nil item with defined selection")
		}
		// SPEC: hash is undefined, so we can put whatever we want here
		hash = "(undefined)"
	} else {
		if index < 0 {
			panic("non-nil item with negative selection")
		}
		hash = item.Hash()
	}

	return SelectResponse{Index: index, Hash: hash}
}

// freezeResponse returns l's frozen representation as a response.
func (l *List) freezeResponse() FreezeResponse {
	return l.Freeze()
}

// Dump handles a dump request.
func (l *List) Dump(dumpCb comm.ResponseCb) {
	// SPEC: see https://universityradioyork.github.io/baps3-spec/protocol/roles/lis
	dumpCb(l.autoModeResponse())
	dumpCb(l.freezeResponse())
	dumpCb(l.selectResponse())
	// TODO(@MattWindsor91): other items in dump
}

//
// Request handling
//

// HandleRequest handles a request for List l.
func (l *List) HandleRequest(replyCb comm.ResponseCb, bcastCb comm.ResponseCb, rbody interface{}) error {
	var err error

	switch b := rbody.(type) {
	case SetAutoModeRequest:
		err = l.handleAutoModeRequest(replyCb, bcastCb, b)
	case SetSelectRequest:
		err = l.handleSelectRequest(replyCb, bcastCb, b)
	case AddItemRequest:
		err = l.handleAddItemRequest(replyCb, bcastCb, b)
	default:
		err = fmt.Errorf("list can't handle this request")
	}

	return err
}

// handleAutoModeRequest handles an automode change request for List l.
func (l *List) handleAutoModeRequest(replyCb comm.ResponseCb, bcastCb comm.ResponseCb, b SetAutoModeRequest) error {
	if l.SetAutoMode(b.AutoMode) {
		bcastCb(l.autoModeResponse())
	}

	// TODO(@MattWindsor91): errors from setting automode?
	return nil
}

// handleSelectRequest handles a selection change request for List l.
func (l *List) handleSelectRequest(replyCb comm.ResponseCb, bcastCb comm.ResponseCb, b SetSelectRequest) error {
	changed, err := l.Select(b.Index, b.Hash)
	if err != nil && changed {
		bcastCb(l.selectResponse())
	}

	return err
}

// handleAddItemRequest handles an item add request for List l.
func (l *List) handleAddItemRequest(replyCb comm.ResponseCb, bcastCb comm.ResponseCb, b AddItemRequest) error {
	err := l.Add(&b.Item, b.Index)
	if err == nil {
		bcastCb(ItemResponse(b))
	}

	return err
}
