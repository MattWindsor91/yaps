package list

import (
	"container/list"
	"fmt"
)

// AutoMode is the type of autoselection modes.
type AutoMode int

const (
	AutoOff AutoMode = iota
	AutoDrop
	AutoNext
	AutoShuffle
)

// String gets the Bifrost name of an AutoMode as a string.
func (a AutoMode) String() string {
	switch a {
	case AutoOff:
		return "off"
	case AutoDrop:
		return "drop"
	case AutoNext:
		return "next"
	case AutoShuffle:
		return "shuffle"
	}
	panic(fmt.Sprintf("unknown automode: %#v", a))
}

// Item is the internal representation of a baps3d list item.
type Item struct {
	// hash is the inserter-supplied unique hash of the item.
	hash string
}

// List is the internal representation of a baps3d list.
// It only maintains the playlist itself: it does not talk to the environment,
// nor does it know anything about what is actually playing.
type List struct {
	// list is the internal linked list representing the playlist.
	// Element type is *Item.
	list *list.List

	// selection is the currently selected index, or -1 if there isn't one.
	selection int

	// autoselect is the current autoselection mode.
	autoselect AutoMode
	// usedHashes is the set of currently spent hashes since the last select.
	// It is used for calculating the next track in AutoShuffle mode.
	usedHashes map[string]struct{}
}

// New creates a new baps3d list.
// The list begins with no selection, an empty list, and autoselect off.
func New() *List {
	return &List{
		list:       list.New(),
		selection:  -1,
		autoselect: AutoOff,
		usedHashes: make(map[string]struct{}),
	}
}


// Count gets the number of items in the list.
func (l *List) Count() int {
	return l.list.Len()
}


// AutoMode gets the current autoselect mode for the given List.
func (l *List) AutoMode() AutoMode {
	return l.autoselect
}

// SetAutoMode changes the current autoselect mode for the given List.
func (l *List) SetAutoMode(mode AutoMode) {
	l.autoselect = mode
	l.clearUsedHashes()
}


// Selection gets the current selection for the given List.
// The selection is returned as a pair of index and possible item.
// If the index is -1, there is no selection, and the item is nil.
func (l *List) Selection() (int, *Item) {
	if l.selection < -1 {
		panic("Selection(): selection negative but not -1")
	}	

	// No selection?
	if l.selection == -1 {
		return -1, nil
	}

	// selection is positive, so we need to walk through the list to find it.
	e := l.list.Front();
	for sel := l.selection; 0 < sel; sel-- {
		// The selection being above the number of items is an internal error.
		if e == nil {
			panic("Selection(): selection out of bounds")
		}

		e = e.Next()
	}

	// In this case, we've found our selected item.
	return l.selection, e.Value.(*Item)
}

// Next advances the selection according to the automode.
// It returns the new selection and a Boolean stating whether the selection changed.
func (l *List) Next() (int, bool) {
	// We can't get the next selection if nothing is selected.
	// TODO(CaptainHayashi): is this true in shuffle mode?
	if l.selection == -1 {
		return -1, false
	}

	switch l.autoselect {
	case AutoOff:
		return l.selection, false
	case AutoDrop:
		l.selection = -1
	case AutoNext:
		l.selection++
		if l.selection >= l.list.Len() {
			l.selection = -1
		}
	case AutoShuffle:
		panic("TODO(CaptainHayashi): implement shuffle")
	}

	return l.selection, true
}

// clearUsedHashes empties the used hash bucket for the given List.
func (l *List) clearUsedHashes() {
	l.usedHashes = make(map[string]struct{})
}
