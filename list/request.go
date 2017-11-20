package list

// This file contains the requests a Controller containing a List understands.
// See 'controller.go' for the controller implementation.
// See 'bifrost.go' for a mapping between these and Bifrost messages.
// See package 'comm' for the higher-level request/response infrastructure.
// - Controllers containing Lists also understand requests from 'comm/request.go'.

// When adding new responses, make sure to add:
// - controller logic in 'controller.go';
// - a parser from messages in 'bifrost.go';
// - an emitter to messages in 'bifrost.go'.

// SetSelectRequest requests a selection change.
type SetSelectRequest struct {
	// Index represents the index to select.
	Index int
	// Hash represents the hash of the item to select.
	// It exists to prevent selection races.
	Hash string
}

// SetAutoModeRequest requests an automode change.
type SetAutoModeRequest struct {
	// AutoMode represents the new AutoMode to use.
	AutoMode AutoMode
}
