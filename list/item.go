package list

// ItemType is the type of types of item.
type ItemType int

const (
	// ItemNone represents a nonexistent item.
	ItemNone ItemType = iota
	// ItemTrack represents a track item.
	// Track items can be selected.
	ItemTrack
	// ItemText represents a textual item.
	// Text items cannot be selected.
	ItemText
)

// String gets the descriptive name of an ItemType as a string.
func (i ItemType) String() string {
	switch i {
	case ItemNone:
		return "none"
	case ItemTrack:
		return "track"
	case ItemText:
		return "text"
	default:
		return "?unknown?"
	}
}

// Item is the internal representation of a baps3d list item.
type Item struct {
	// hash is the inserter-supplied unique hash of the item.
	hash string
	// payload is the data component of the item.
	payload string
	// itype is the type of tie item.
	itype ItemType
}

// NewTrack creates a new track-type item.
func NewTrack(hash, path string) *Item {
	return &Item{hash, path, ItemTrack}
}

// NewText creates a new text-type item.
func NewText(hash, contents string) *Item {
	return &Item{hash, contents, ItemText}
}

// Type returns the type of the Item.
func (i *Item) Type() ItemType {
	return i.itype
}

// Payload returns the payload of the Item.
func (i *Item) Payload() string {
	return i.payload
}

// Hash returns the hash of the Item.
func (i *Item) Hash() string {
	return i.hash
}
