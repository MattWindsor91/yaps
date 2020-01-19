package comm_test

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/UniversityRadioYork/baps3d/comm"
)

type testState struct{}

/*
Dummy requests and responses
*/

type knownDummyRequest struct {
	// True if the dummy response should be broadcast.
	Broadcast bool
}
type knownDummyResponse struct{}

/*
Controllable implementation
*/

func (*testState) RoleName() string {
	return "test"
}

func (*testState) Dump(comm.ResponseCb) {}

func (*testState) HandleRequest(replyCb, bcastCb comm.ResponseCb, rbody interface{}) error {
	switch b := rbody.(type) {
	case knownDummyRequest:
		var cb comm.ResponseCb
		if b.Broadcast {
			cb = bcastCb
		} else {
			cb = replyCb
		}

		cb(knownDummyResponse{})
		return nil
	default:
		return fmt.Errorf("unknown request")
	}
}

type testStateWithParser struct {
	testState
}

/*
BifrostParser implementation for testStateWithParser
*/

func (*testStateWithParser) ParseBifrostRequest(word string, _ []string) (interface{}, error) {
	return nil, comm.UnknownWord(word)
}

func (*testStateWithParser) EmitBifrostResponse(string, interface{}, chan<- bifrost.Message) error {
	return nil
}

/*
Test helpers
*/

func testWithController(s comm.Controllable, f func(context.Context, *comm.Client, *testing.T), t *testing.T) {
	t.Helper()

	innerCtx, cancel := context.WithCancel(context.Background())

	ctl, client := comm.NewController(s)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		ctl.Run(innerCtx)
		cancel()
		wg.Done()
	}()

	f(innerCtx, client, t)

	if err := client.Shutdown(innerCtx); err != nil {
		t.Errorf("error shutting client down after test: %s", err.Error())
	}
	wg.Wait()
}

/*
Test functions
*/

// TestClient_Send_Reply tests using Client.Send to send a known request with
// a unicast reply.
func TestClient_Send_Reply(t *testing.T) {
	f := func(ctx context.Context, c *comm.Client, t *testing.T) {
		reply := make(chan comm.Response)

		rq := comm.Request{
			Origin: comm.RequestOrigin{Tag: "test1", ReplyTx: reply},
			Body:   knownDummyRequest{},
		}

		if !c.Send(ctx, rq) {
			t.Fatal("controller shut down before we could send test request")
		}

		checkReply := func(slot, typename string) {
			rr, rrok := <-reply
			if !rrok {
				t.Fatalf("reply channel closed after %s response", slot)
			}
			if rr.Broadcast {
				t.Errorf("%s response erroneously marked as broadcast", slot)
			}
			if rr.Origin == nil {
				t.Errorf("%s response erroneously has no origin", slot)
			} else if rr.Origin.Tag != "test1" {
				t.Errorf("%s response has wrong tag: got %s", slot, rr.Origin.Tag)
			}
			rrtype := reflect.TypeOf(rr.Body).String()
			if rrtype != typename {
				t.Fatalf("unexpected %s response type: want %s, got %s", slot, typename, rrtype)
			}
		}
		checkReply("first", "comm_test.knownDummyResponse")
		checkReply("second", "comm.AckResponse")
	}
	testWithController(&testState{}, f, t)
}

// TestClient_Bifrost_NoBifrostParser tests Client.Bifrost's behaviour when its
// parent Controller's inner state doesn't understand Bifrost.
func TestClient_Bifrost_NoBifrostParser(t *testing.T) {
	f := func(ctx context.Context, cli *comm.Client, t *testing.T) {
		bf, bfc, err := cli.Bifrost(ctx)
		if err == nil {
			t.Errorf("expected an error")
		}
		if err != comm.ErrControllerCannotSpeakBifrost {
			t.Errorf("incorrect error sent")
		}

		if bf != nil {
			t.Error("received non-nil Bifrost from failing Bifrost() call")
		}

		if bfc != nil {
			t.Error("received non-nil BifrostClient from failing Bifrost() call")
		}
	}
	testWithController(&testState{}, f, t)
}

// TestClient_Bifrost_BifrostParser tests Client.Bifrost's behaviour when its
// parent Controller's inner state understands Bifrost.
func TestClient_Bifrost_BifrostParser(t *testing.T) {
	f := func(ctx context.Context, cli *comm.Client, t *testing.T) {
		bf, bfc, err := cli.Bifrost(ctx)
		if err != nil {
			t.Errorf("got unexpected error: %s", err.Error())
		}

		if bf == nil {
			t.Error("got nil Bifrost from passing Bifrost() call")
		}

		if bfc == nil {
			t.Error("got nil BifrostClient from passing Bifrost() call")
		}
	}
	testWithController(&testStateWithParser{}, f, t)
}

// TestClient_Shutdown tests Client.Shutdown's behaviour.
func TestClient_Shutdown(t *testing.T) {
	f := func(ctx context.Context, c *comm.Client, t *testing.T) {
		if err := c.Shutdown(ctx); err != nil {
			t.Fatalf("unexpected error on first shutdown: %s", err.Error())
		}
		// Sends should terminate but fail.
		// This test isn't robust: it could be that broken implementations of
		// Shutdown doesn't always fail to shut down before returning.
		reply := make(chan comm.Response)
		if c.Send(ctx, comm.Request{
			Origin: comm.RequestOrigin{
				Tag:     "",
				ReplyTx: reply,
			},
			Body: knownDummyRequest{},
		}) {
			t.Error("send to shut-down Client erroneously succeeded")
		}
		// Double shutdowns shouldn't trip errors or diverge.
		if err := c.Shutdown(ctx); err != nil {
			t.Errorf("unexpected error on second shutdown: %s", err.Error())
		}
	}
	testWithController(&testState{}, f, t)
}

// TestClient_CopyBeforeShutdown tests what happens when we shutdown a
// controller with a copied client.
func TestClient_CopyBeforeShutdown(t *testing.T) {
	f := func(ctx context.Context, c *comm.Client, t *testing.T) {
		c2, err := c.Copy(ctx)
		if err != nil {
			t.Fatalf("unexpected error on copy: %s", err.Error())
		}

		if err := c.Shutdown(ctx); err != nil {
			t.Fatalf("unexpected error on original shutdown: %s", err.Error())
		}

		// The second client shouldn't be taking requests.
		reply := make(chan comm.Response)
		if c2.Send(ctx, comm.Request{
			Origin: comm.RequestOrigin{
				Tag:     "",
				ReplyTx: reply,
			},
			Body: knownDummyRequest{},
		}) {
			t.Error("send to shut-down Client copy erroneously succeeded")
		}

		// The second client shouldn't error on a second shutdown.
		if err := c2.Shutdown(ctx); err != nil {
			t.Fatalf("unexpected error on copy shutdown: %s", err.Error())
		}
	}
	testWithController(&testState{}, f, t)
}

// TestClient_CopyAfterShutdown tests Client.Copy's behaviour on a shut-down client.
func TestClient_CopyAfterShutdown(t *testing.T) {
	f := func(ctx context.Context, c *comm.Client, t *testing.T) {
		if err := c.Shutdown(ctx); err != nil {
			t.Fatalf("unexpected error on shutdown: %s", err.Error())
		}
		c2, err := c.Copy(ctx)
		if err == nil {
			t.Fatalf("didn't get error when Copying on a shutdown controller")
		}
		if err != comm.ErrControllerShutDown {
			t.Fatalf("got wrong error when Copying on a shutdown controller: %s", err.Error())
		}
		if c2 != nil {
			t.Fatalf("got non-nil Client when Copying on a shutdown controller")
		}
	}
	testWithController(&testState{}, f, t)
}
