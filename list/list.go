package list

import (
	"container/list"
	"fmt"
	"time"
	"math/rand"	
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
	default:
		return "?unknown?"
	}
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
	// rng is the random number generator for autoshuffling.
	rng *rand.Rand
	// usedHashes is the set of currently spent hashes since the last select.
	// It is used for calculating the next track in AutoShuffle mode.
	usedHashes map[string]struct{}
}

// New creates a new baps3d list.
// The list begins with no selection, an empty list, and autoselect off.
func New() *List {
	// Hopefully, the current time is an ok seed.
	// This just needs to be 'random enough', not foolproof
	src := rand.NewSource(time.Now().Unix())

	return &List{
		list:       list.New(),
		selection:  -1,
		autoselect: AutoOff,
		rng:        rand.New(src),
		usedHashes: make(map[string]struct{}),
	}
}

// Add adds an Item to a list.
// It will fail if there is already an Item with the same hash enqueued.
func (l *List) Add(item *Item, i int) error {
	if j, _ := l.ItemWithHash(item.Hash()); j > -1 {
		return fmt.Errorf("List.Add(): duplicate hash %s at index %i", item.Hash(), j)
	}

	// Adding an item on or before the current selection moves it down one.
	if i <= l.selection {
		l.selection++
	}

	// We have to handle the 'front of list' situation specially:
	// all the other ones expect a predecessor element.
	if i == 0 {
		l.list.PushFront(item)
		return nil
	}

	if e := l.elementWithIndex(i - 1); e != nil {
		l.list.InsertAfter(item, e)
		return nil
	}

	// There was no predecessor, and index is not 0, so we've overshot
	return fmt.Errorf("Tried to insert element at index %d when there are only %d item(s)", i, l.Count())
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
	// If we've _just_ changed to shuffle mode, prepare the state for it.
	if mode == AutoShuffle && l.autoselect != AutoShuffle {
		l.clearUsedHashes()
	}

	l.autoselect = mode
}

// elementWithIndex tries to find the linked list node with the given index.
// It returns nil if one couldn't be found.
func (l *List) elementWithIndex(i int) (*list.Element) {
	// Keep going until we either run out of items, or find the right index.
	// This is O(n), but the lists will usually be quite small anyway.
	e := l.list.Front();
	for j := 0; e != nil && j != i; j-- {
		e = e.Next()
	}
	return e
}

// ItemWithIndex tries to find the item with the given index.
// The result is returned as a pair of 'ok' flag and possible item.
// If the flag is false, there is no item with that index, and the item is nil.
func (l *List) ItemWithIndex(i int) (bool, *Item) {
	if e := l.elementWithIndex(i); e != nil {
		return true, e.Value.(*Item)
	}
	return false, nil
}

// elementWithIndex tries to find the linked list node with the given index.
// It returns (-1, nil) if one couldn't be found.
func (l *List) elementWithHash(hash string) (int, *list.Element) {
	// Keep going until we either run out of items, or find ours.
	// This is O(n), but the lists will usually be quite small anyway.
	i := 0
	for e := l.list.Front(); e != nil; e = e.Next() {
		item := e.Value.(*Item)
		if item.Hash() == hash {
			return i, e
		}
		i++
	}

	// We didn't find the item (the case where we did is handled in the loop).
	return -1, nil
}

// ItemWithHash tries to find the item with the given hash.
// The result is returned as a pair of index and possible item.
// If the index is -1, there is no item with that hash, and the item is nil.
func (l *List) ItemWithHash(hash string) (int, *Item) {
	if i, e := l.elementWithHash(hash); e != nil {
		return i, e.Value.(*Item)
	}
	return -1, nil
}

// Selection gets the current selection for the given List.
// The selection is returned as a pair of index and possible item.
// If the index is -1, there is no selection, and the item is nil.
func (l *List) Selection() (int, *Item) {
	// No selection?
	if l.selection == -1 {
		return -1, nil
	}

	if ok, item := l.ItemWithIndex(l.selection); ok {
		return l.selection, item
	}

	// The selection not being found is an internal error.
	panic("Selection(): selection not in list")
}

// Next advances the selection according to the automode.
// It returns the new selection and a Boolean stating whether the selection changed.
func (l *List) Next() (int, bool) {
	e := l.elementWithIndex(l.selection)
	// We can't get the next selection if nothing is selected.
	// TODO(CaptainHayashi): is this true in shuffle mode?
	if e == nil {
		return -1, false
	}

	ni, nh := l.chooseNext(l.selection, e)
	l.selection = ni
	return ni, nh != e.Value.(*Item).Hash()
}

// chooseNext chooses the next selection based on the given previous selection element.
func (l *List) chooseNext(i int, prev *list.Element) (int, string) {
	switch l.autoselect {
	case AutoOff:
		return i, prev.Value.(*Item).hash
	case AutoDrop:
		return -1, ""
	case AutoNext:
		if e := prev.Next(); e != nil {
			return i + 1, e.Value.(*Item).Hash()
		}
		return -1, ""
	case AutoShuffle:
		return l.shuffleChoose()
	}

	// TODO: error here?
	return -1, ""
}

// clearUsedHashes empties the used hash bucket for the given List.
func (l *List) clearUsedHashes() {
	l.usedHashes = make(map[string]struct{})
}


// shuffleChoose selects a random item from the playlist.
// It will not select an item whose hash is in the used hash bucket.
// It returns a the index and hash.
func (l *List) shuffleChoose() (int, string) {
	// First, work out which items are available.
	/* TODO(CaptainHayashi): this is slow, but guaranteed to terminate.
	   Randomly choosing a hash then checking it for previous play would be faster
	   in some cases, but could technically never terminate. */
	count := 0
	unpickedH := make([]string, l.list.Len())
	unpickedI := make([]int, l.list.Len())
	i := 0
	for e := l.list.Front(); e != nil; e = e.Next() {
		le := e.Value.(*Item)
		lh := le.Hash()
		if _, in := l.usedHashes[lh]; !in {
			unpickedH[count] = lh
			unpickedI[count] = i
			count++
		}
		i++
	}

	/* If we didn't find anything, we're done with this shuffle.
	   Prepare a new one. */
	if count == 0 {
		l.clearUsedHashes()
		return -1, ""
	}

	s := l.rng.Intn(count)
	l.usedHashes[unpickedH[s]] = struct{}{}
	return unpickedI[s], unpickedH[s]
}
