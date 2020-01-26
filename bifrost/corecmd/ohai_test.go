package corecmd

import (
	"fmt"
	"testing"

	"github.com/UniversityRadioYork/baps3d/bifrost/msgproto"
)

// ExampleParseOhaiResponse is a testable example for ParseOhaiResponse.
func ExampleParseOhaiResponse() {
	m := msgproto.NewMessage(msgproto.TagBcast, RsOhai).AddArgs("test-0.2.0", "example-42.0.0")
	if o, err := ParseOhaiResponse(m); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Protocol:", o.ProtocolVer)
		fmt.Println("Server:", o.ServerVer)
	}

	// Output:
	// Protocol: test-0.2.0
	// Server: example-42.0.0
}

var ohaiResponseRoundTripCases = []OhaiResponse{
	{
		ProtocolVer: "test-0.2.0",
		ServerVer:   "example-42.0.0",
	},
	{
		ProtocolVer: "foobar-1.8.0",
		ServerVer:   "baz-0.0.1",
	},
}

// TestParseOhaiResponse_roundTrip checks that parsing the result of OhaiResponse's Message method returns a similar
// OhaiResponse.
func TestParseOhaiResponse_roundTrip(t *testing.T) {
	for _, c := range ohaiResponseRoundTripCases {
		m := c.Message(msgproto.TagBcast)

		if got, err := ParseOhaiResponse(m); err != nil {
			t.Errorf("parse error: %v", err)
		} else if got.ProtocolVer != c.ProtocolVer {
			t.Errorf("got protocol %q; want %q", got.ProtocolVer, c.ProtocolVer)
		} else if got.ServerVer != c.ServerVer {
			t.Errorf("got server %q; want %q", got.ServerVer, c.ServerVer)
		}
	}
}

// TestParseOhaiResponse_wordError checks that ParseOhaiResponse handles word errors correctly.
func TestParseOhaiResponse_wordError(t *testing.T) {
	// Hopefully this is a decently representative set of variations on OHAI.
	cases := []string{
		"",
		"ohai",
		"OH",
		"Ohai",
		"OHAITHAR",
	}

	for _, word := range cases {
		m := msgproto.NewMessage(msgproto.TagBcast, word).AddArgs("bifrost-0.3.9", "bigboy-4.8.8.4")

		_, err := ParseOhaiResponse(m)
		testParserWordError(t, err, RsOhai, word)
	}
}

// TestParseOhaiResponse_arityErrors checks that ParseOhaiResponse handles arity errors correctly.
func TestParseOhaiResponse_arityErrors(t *testing.T) {
	cases := [][]string{
		{},
		{"bifrost-0.0.1"},
		{"bifrost-0.0.1", "itones-4.10.1998", "but wait there's more!"},
	}

	for _, args := range cases {
		m := msgproto.NewMessage(msgproto.TagBcast, RsOhai).AddArgs(args...)

		_, err := ParseOhaiResponse(m)
		testArgUnpackError(t, "ParseOhaiResponse", 2, len(args), err)
	}
}
