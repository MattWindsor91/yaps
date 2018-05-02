package comm_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/UniversityRadioYork/baps3d/comm"
)

type testState struct{}

/*
Dummy requests and responses
*/

type knownDummyRequest struct{}
type unknownDummyRequest struct{}

/*
Controllable implementation
*/

func (*testState) RoleName() string {
	return "test"
}

func (*testState) Dump(comm.ResponseCb) {}

func (*testState) HandleRequest(replyCb, bcastCb comm.ResponseCb, rbody interface{}) error {
	return fmt.Errorf("unknown request")
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

func testWithController(s comm.Controllable, f func(*comm.Client, *testing.T), t *testing.T) {
	t.Helper()

	ctl, client := comm.NewController(s)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		ctl.Run()
		wg.Done()
	}()

	f(client, t)

	client.Shutdown()
	wg.Wait()
}

/*
Test functions
*/

// TestClient_Bifrost_NoBifrostParser tests Client.Bifrost's behaviour when its
// parent Controller's inner state doesn't understand Bifrost.
func TestClient_Bifrost_NoBifrostParser(t *testing.T) {
	f := func(cli *comm.Client, t *testing.T) {
		bf, bfc, err := cli.Bifrost()
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
	f := func(cli *comm.Client, t *testing.T) {
		bf, bfc, err := cli.Bifrost()
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
	f := func(c *comm.Client, t *testing.T) {
		c.Shutdown()
		// Sends should terminate but fail.
		// This test isn't robust: it could be that broken implementations of
		// Shutdown doesn't always fail to shut down before returning.
		reply := make(chan comm.Response)
		if c.Send(comm.Request{
			Origin: comm.RequestOrigin{
				Tag:     "",
				ReplyTx: reply,
			},
			Body: knownDummyRequest{},
		}) {
			t.Error("send to shut-down Client erroneously succeeded")
		}
		// Double shutdowns shouldn't trip errors or diverge.
		c.Shutdown()
	}
	testWithController(&testState{}, f, t)
}
