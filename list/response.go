package list

// File response.go contains the responses a Controller containing a List can send.
// - See `list/controller.go` for the controller implementation.
// - See `list/bifrost.go` for a mapping between these and Bifrost messages.
// - See package 'comm' for the higher-level request/response infrastructure.
//   Controllers containing Lists can also send responses from `comm/response.go`.

// When adding new responses, make sure to add:
// - controller logic in 'controller.go';
// - a parser from messages in 'bifrost.go';
// - an emitter to messages in 'bifrost.go'.

// AutoModeResponse announces a change in AutoMode.
type AutoModeResponse struct {
	// AutoMode represents the new AutoMode.
	AutoMode AutoMode
}

// SelectResponse announces a change in selection.
type SelectResponse struct {
	// Index represents the selected index.
	Index int
	// Hash represents the selected item's hash.
	Hash string
}

// FreezeResponse announces a snapshot of the entire list.
type FreezeResponse []Item

// ItemResponse announces the presence of a single list item.
type ItemResponse struct {
	// Index is the index of the item in the list.
	Index int
	// Item is the item itself.
	Item Item
}
