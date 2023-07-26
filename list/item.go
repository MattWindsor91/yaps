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

// Item is the internal representation of a yaps list item.
type Item struct {
	// hash is the inserter-supplied unique hash of the item.
	hash string
	// payload is the data component of the item.
	payload string
	// itype is the type of the item.
	itype ItemType
}

// NewItem creates a new item with the given hash, payload, and item type.
func NewItem(itype ItemType, hash, payload string) *Item {
	return &Item{hash, payload, itype}
}

// NewTrack creates a new track-type item.
func NewTrack(hash, path string) *Item {
	return NewItem(ItemTrack, hash, path)
}

// NewText creates a new text-type item.
func NewText(hash, contents string) *Item {
	return NewItem(ItemText, hash, contents)
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

// IsSelectable returns whether or not the Item i can be selected.
func (i *Item) IsSelectable() bool {
	return i.itype != ItemText
}
