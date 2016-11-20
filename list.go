package main

import "container/list"

// AutoMode is the type of autoselection modes.
type AutoMode int

const (
	AutoOff AutoMode = iota
	AutoDrop
	AutoNext
	AutoShuffle
)

// List is the internal representation of a baps3d list.
// It only maintains the playlist itself: it does not talk to the environment,
// nor does it know anything about what is actually playing.
type List struct {
	/* list is the internal linked list representing the playlist. */
	list *list.List

	/* selection is the currently selected index, or -1 if there isn't one. */
	selection int

	/* autoselect is the current autoselection mode. */
	autoselect AutoMode
	/* usedHashes is the set of currently spent hashes since the last select.
	   It is used for calculating the next track in AutoShuffle mode. */
	usedHashes map[string]struct{}
}

// NewList creates a new baps3d list.
func NewList() *List {
	return &List{
		list:       list.New(),
		selection:  -1,
		autoselect: AutoOff,
		usedHashes: make(map[string]struct{}),
	}
}

// SetAutoMode changes the current autoselect mode for the given List.
func (l *List) SetAutoMode(mode AutoMode) {
	l.autoselect = mode
	l.clearUsedHashes()
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
