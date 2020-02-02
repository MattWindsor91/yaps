package msgproto

import "github.com/google/uuid"

const (
	// TagBcast is the tag used for broadcasts.
	TagBcast string = "!"

	// TagUnknown is the tag used for when we don't know the right tag to use.
	TagUnknown string = "?"
)

// NewTag generates a pseudorandom tag.
// This tag should be unique enough to distinguish any communications sent using it from others,
// including those made by the same client or server.
func NewTag() (string, error) {
	// this should be Version 1
	u, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
