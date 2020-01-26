package msgproto

// File test_helpers.go contains helper functions for testing parts of the Bifrost message protocol.

import (
	"reflect"
	"testing"
)

// AssertMessagesEqual checks whether two messages (expected and actual) are equal up to packed representation.
// It throws a test failure if not, or if either message fails to pack.
// The parameter in should give a brief description of the context of this assertion.
func AssertMessagesEqual(t *testing.T, in string, got, want *Message) {
	t.Helper()

	var (
		wp, gp []byte
		err    error
	)
	if wp, err = want.Pack(); err != nil {
		t.Errorf("%s: expected message failed to pack: %v", in, err)
		return
	}
	if gp, err = got.Pack(); err != nil {
		t.Errorf("%s: actual message failed to pack: %v", in, err)
		return
	}
	if !reflect.DeepEqual(gp, wp) {
		t.Errorf("%s: got message %s, want %s", in, string(gp), string(wp))
	}
}
