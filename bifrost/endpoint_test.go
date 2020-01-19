package bifrost

import (
	"context"
	"testing"
)

// File bifrost/endpoint_test.go contains tests for the Endpoint struct.

// Tests that a pair of endpoints produced by NewEndpointPair connect to each other correctly.
func TestNewEndpointPair_TxRx(t *testing.T) {
	l, r := NewEndpointPair()

	testEndpointTxRx(t, l.Tx, r.Rx)
	testEndpointTxRx(t, r.Tx, l.Rx)
}

// Tests one side of an endpoint pair Tx/Rx connection.
func testEndpointTxRx(t *testing.T, tx chan<- Message, rx <-chan Message) {
	t.Helper()

	msg := NewMessage("foo", "bar").AddArg("baz")
	go func() { tx <- *msg }()
	msg2 := <-rx

	AssertMessagesEqual(t, msg, &msg2)
}

func TestEndpoint_Send(t *testing.T) {
	l, r := NewEndpointPair()
	ctx, cancel := context.WithCancel(context.Background())

	msg := NewMessage("!", "jam").AddArg("on").AddArg("toast")

	go func() {
		if !l.Send(ctx, *msg) {
			t.Error("send failed unexpectedly")
		}
	}()

	msg2 := <-r.Rx
	AssertMessagesEqual(t, msg, &msg2)

	// After cancelling, sends should fail.
	cancel()

	go func() {
		if l.Send(ctx, *msg) {
			t.Error("send succeeded unexpectedly")
		}
	}()

}