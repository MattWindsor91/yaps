package comm_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/UniversityRadioYork/baps3d/comm"
)

type TestState struct{}

// Controllable implementation

func (*TestState) RoleName() string {
	return "test"
}

func (*TestState) Dump(comm.ResponseCb) {}

func (*TestState) HandleRequest(replyCb, bcastCb comm.ResponseCb, rbody interface{}) error {
	return fmt.Errorf("unknown request")
}

type TestStateWithParser struct {
	TestState
}

// TestClient_Bifrost_NoBifrostParser tests Client.Bifrost's behaviour when its
// parent Controller's inner state doesn't understand Bifrost.
func TestClient_Bifrost_NoBifrostParser(t *testing.T) {
	s := TestState{}
	cnt, cli := comm.NewController(&s)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		cnt.Run()
		wg.Done()
	}()

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

	cli.Shutdown()
	wg.Wait()
}
