package msgproto

// File test_helpers.go contains helper functions for testing parts of the Bifrost message protocol.

import (
	"reflect"
	"testing"
)

// AssertMessagesEqual checks whether two messages (expected and actual) are equal up to packed representation.
// It throws a test failure if not, or if either message fails to pack.
func AssertMessagesEqual(t *testing.T, expected, actual *Message) {
	var (
		ep, ap []byte
		err    error
	)
	if ep, err = expected.Pack(); err != nil {
		t.Errorf("expected message failed to pack: %v", err)
		return
	}
	if ap, err = actual.Pack(); err != nil {
		t.Errorf("actual message failed to pack: %v", err)
		return
	}
	if !reflect.DeepEqual(ep, ap) {
		t.Errorf("expected message %s, got %s", string(ep), string(ap))
	}
}
