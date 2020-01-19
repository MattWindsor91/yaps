package bifrost

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/jordwest/mock-conn"
)

// TestIoClient_Run_Tx tests the running of an IoClient by transmitting several raw Bifrost messages down a mock TCP
// connection and seeing whether they come through the Bifrost RX channel as properly parsed messages.
func TestIoClient_Run_Tx(t *testing.T) {
	var wg sync.WaitGroup
	endp, tcp := runMockIoClient(t, context.Background(), &wg)

	cases := []struct {
		input    string
		expected *Message
	}{
		{ "! IAMA saucepan", NewMessage("!", "IAMA").AddArg("saucepan") },
		{ "f00f STOP 'hammer time'", NewMessage("f00f", "STOP").AddArg("hammer time") },
		{ "? foobar 'qu'u'x' 'x'y'z'z'y'", NewMessage("?", "foobar").AddArg("quux").AddArg("xyzzy")},
	}

	for _, c := range cases {
		if _, err := fmt.Fprintln(tcp, c.input); err != nil {
			t.Fatalf("error sending raw message: %v", err)
		}

		m := <- endp.Rx
		mp := strings.TrimSpace(m.String())
		ep := strings.TrimSpace(c.expected.String())
		if mp != ep {
			t.Errorf("expected message [%s], got message [%s]", ep, mp)
		}
	}

	if err := tcp.Close(); err != nil {
		t.Fatalf("tcp close error: %v", err)
	}
	wg.Wait()
}

// TestIoClient_Run_Rx tests the running of an IoClient by sending several Bifrost messages down its Rx channel, and
// making sure the resulting traffic through an attached mock TCP connection matches up.
func TestIoClient_Run_Rx(t *testing.T) {
	var wg sync.WaitGroup
	endp, tcp := runMockIoClient(t, context.Background(), &wg)
	rd := bufio.NewReader(tcp)

	cases := []struct {
		input    *Message
		expected string
	}{
		{ NewMessage("!", "IAMA").AddArg("chest of drawers"), "! IAMA 'chest of drawers'" },
		{ NewMessage("?", "make").AddArg("me").AddArg("a 'sandwich'"), `? make me 'a '\''sandwich'\'''` },
		{ NewMessage("i386", "blorf"), "i386 blorf" },
	}


	// Send all in one block, and later receive all in one block, to make it easier to handle any IoClient errors.
	for _, c := range cases {
		var (
			s string
			err error
		)

		endp.Tx <- *c.input
		if s, err = rd.ReadString('\n'); err != nil {
			t.Fatalf("tcp error: %v", err)
		}
		s = strings.TrimSpace(s)
		if c.expected != s {
			t.Errorf("expected [%s], got [%s]", c.expected, s)
		}
	}

	if err := tcp.Close(); err != nil {
		t.Fatalf("tcp close error: %v", err)
	}
	wg.Wait()
}


// runMockIoClient makes and sets-running an IoClient with a simulated TCP connection.
// It returns an Endpoint and io.ReadWriteCloser that can be used to manipulate both ends of the mock connection.
// It also sets up a goroutine for tracking errors from the IoClient.
func runMockIoClient(t *testing.T, ctx context.Context, wg *sync.WaitGroup) (*Endpoint, io.ReadWriteCloser) {
	t.Helper()

	wg.Add(2)

	ic, bfe, conn := makeMockIoClient(t)

	errCh := make(chan error)

	go func() {
		ic.Run(ctx, errCh)
		wg.Done()
	}()
	go func() {
		for e := range errCh {
			if errors.Is(e, HungUpError) {
				bfe.Close()
			} else if !errors.Is(e, io.EOF) {
				t.Errorf("ioclient error: %v", e)
			}
		}
		wg.Done()
	}()

	return bfe, conn
}

// makeMockIoClient constructs an IoClient with a simulated TCP connection.
// It returns the client itself, the Bifrost endpoint for inspecting the messages sent and received from the IoClient,
// and the fake TCP/IP connection simulating a remote client.
func makeMockIoClient(t *testing.T) (*IoClient, *Endpoint, *mock_conn.End) {
	t.Helper()

	conn := mock_conn.NewConn()
	bfc, bfe := NewEndpointPair()
	ic := IoClient{Conn: conn.Server, Bifrost: bfc}
	return &ic, bfe, conn.Client
}
