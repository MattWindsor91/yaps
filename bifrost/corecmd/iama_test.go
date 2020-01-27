package corecmd

import (
	"fmt"
	"testing"

	"github.com/UniversityRadioYork/baps3d/bifrost/msgproto"
)

// ExampleParseIamaResponse is a testable example for ParseIamaResponse.
func ExampleParseIamaResponse() {
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

var iamaResponseRoundTripCases = []IamaResponse{
	IamaResponse{"list"}, IamaResponse{"player/file"}, IamaResponse{"x/foobar"},
}

// TestParseIamaResponse_roundTrip checks that parsing the result of OhaiResponse's Message method returns a similar
// IamaResponse.
func TestParseIamaResponse_roundTrip(t *testing.T) {
	for _, c := range iamaResponseRoundTripCases {
		m := c.Message(msgproto.TagBcast)

		if got, err := ParseIamaResponse(m); err != nil {
			t.Errorf("parse error: %v", err)
		} else if got.Role != c.Role {
			t.Errorf("got role %q; want %q", got.Role, c.Role)
		}
	}
}

// TestParseIamaResponse_wordError checks that ParseIamaResponse handles word errors correctly.
func TestParseIamaResponse_wordError(t *testing.T) {
	// Hopefully this is a decently representative set of variations on OHAI.
	cases := []string{
		"",
		"iama",
		"IAM",
		"Iama",
		"IAMAI",
	}

	for _, word := range cases {
		m := msgproto.NewMessage(msgproto.TagBcast, word).AddArgs("test")

		_, err := ParseIamaResponse(m)
		testParserWordError(t, err, RsIama, word)
	}
}

// TestParseIamaResponse_arityErrors checks that ParseIamaResponse handles arity errors correctly.
func TestParseIamaResponse_arityErrors(t *testing.T) {
	cases := [][]string{
		{},
		{"player/file", "but wait there's more!"},
	}

	for _, args := range cases {
		m := msgproto.NewMessage(msgproto.TagBcast, RsIama).AddArgs(args...)

		_, err := ParseIamaResponse(m)
		testArgUnpackError(t, "ParseIamaResponse", 1, len(args), err)
	}
}
