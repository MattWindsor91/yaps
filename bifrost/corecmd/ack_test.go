package corecmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/UniversityRadioYork/baps3d/bifrost/msgproto"
)

// ExampleErrorAck is a testable example for ErrorAck.
func ExampleErrorAck() {
	err := ArityError{Got: 3, Min: 1, Max: 2}
	ack := ErrorAck(err)
	fmt.Println("Status:", ack.Status)
	fmt.Println("Description:", ack.Description)

	// Output:
	// Status: WHAT
	// Description: message has 3 arguments, want 1-2
}

// ExampleParseAckResponse is a testable example for ParseAckResponse.
func ExampleParseAckResponse() {
	m := msgproto.NewMessage(msgproto.TagBcast, RsAck).AddArgs(WordWhat, "description here")
	if ack, err := ParseAckResponse(m); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Status:", ack.Status)
		fmt.Println("Description:", ack.Description)
	}

	// Output:
	// Status: WHAT
	// Description: description here
}

var ackResponseMessageCases = []struct {
	input AckResponse
	tag   string
	want  *msgproto.Message
}{
	// Making sure AckOk is what we think it is.
	{AckOk,
		msgproto.TagBcast,
		msgproto.NewMessage(msgproto.TagBcast, RsAck).AddArgs(WordOk, OkDescription),
	},
	// Testing a WHAT error.
	{
		AckResponse{
			Status:      StatusWhat,
			Description: "u wot, m8?",
		},
		msgproto.TagUnknown,
		msgproto.NewMessage(msgproto.TagUnknown, RsAck).AddArgs(WordWhat, "u wot, m8?"),
	},
	// Testing a FAIL error.
	{
		AckResponse{
			Status:      StatusFail,
			Description: "computer says no",
		},
		"f00f",
		msgproto.NewMessage("f00f", RsAck).AddArgs(WordFail, "computer says no"),
	},
}

var ackResponseRoundTripCases = []AckResponse{
	AckOk,
	// Testing a WHAT error.
	{
		Status:      StatusWhat,
		Description: "u wot, m8?",
	},
	// Testing a FAIL error.
	{
		Status:      StatusFail,
		Description: "computer says no",
	},
}

var errorAckCases = []struct {
	input  error
	status Status
}{
	{nil, StatusOk},
	{WordError{
		Got:  "ack",
		Want: "ACK",
	}, StatusWhat},
	{failError{}, StatusFail},
	{errors.New("anonymous"), StatusFail},
}

// TestAckResponse_Message tests applying the Message method to various AckResponses.
func TestAckResponse_Message(t *testing.T) {
	for _, c := range ackResponseMessageCases {
		got := c.input.Message(c.tag)
		gotStr := fmt.Sprintf("(%q).Message(%s)", c.input, c.tag)
		msgproto.AssertMessagesEqual(t, gotStr, got, c.want)
	}
}

// TestParseAckResponse_roundTrip checks that parsing the result of AckResponse's Message method returns a similar
// AckResponse.
func TestParseAckResponse_roundTrip(t *testing.T) {
	for _, c := range ackResponseRoundTripCases {
		m := c.Message(msgproto.TagBcast)

		if got, err := ParseAckResponse(m); err != nil {
			t.Errorf("parse error: %v", err)
		} else if got.Status != c.Status {
			t.Errorf("got status %s; want %s", got.Status, c.Status)
		} else if got.Description != c.Description {
			t.Errorf("got description %s; want %s", got.Description, c.Description)
		}
	}
}

// TestParseAckResponse_wordError checks that ParseAckResponse handles word errors correctly
func TestParseAckResponse_wordError(t *testing.T) {
	// Hopefully this is a decently representative set of variations on ACK.
	cases := []string{
		"",
		"ack",
		"AC",
		"Ack",
		"ACKERMAN",
	}

	for _, word := range cases {
		m := msgproto.NewMessage(msgproto.TagBcast, word).AddArgs(WordOk, "success")

		_, err := ParseAckResponse(m)
		testParserWordError(t, err, RsAck, word)
	}
}

// TestParseAckResponse_arityErrors checks that ParseAckResponse handles arity errors correctly.
func TestParseAckResponse_arityErrors(t *testing.T) {
	cases := [][]string{
		{},
		{WordOk},
		{WordOk, "success", "but wait there's more!"},
	}

	for _, args := range cases {
		m := msgproto.NewMessage(msgproto.TagBcast, RsAck).AddArgs(args...)

		_, err := ParseAckResponse(m)
		testArgUnpackError(t, "ParseAckResponse", 2, len(args), err)
	}
}

// TestParseAckResponse_statusErrors checks that ParseAckResponse handles status errors correctly.
func TestParseAckResponse_statusErrors(t *testing.T) {
	cases := []string{"", "o", "okay", "wha", "FAILED"}
	for _, c := range cases {
		m := msgproto.NewMessage(msgproto.TagBcast, RsAck).AddArgs(c, "success")

		_, err := ParseAckResponse(m)
		if err == nil {
			t.Errorf("no error when parsing bad-status ack (%s)", m)
			continue
		}

		checkStatusError(t, err, c)
	}
}

// TestErrorAck tests ErrorAck with various error inputs.
func TestErrorAck(t *testing.T) {
	for _, c := range errorAckCases {
		got := ErrorAck(c.input)

		if status := got.Status; status != c.status {
			t.Errorf("ErrorAck(%q).Status = %s; want %s", c.input, got.Status, c.status)
		}

		var wantDesc string
		if c.input == nil {
			wantDesc = OkDescription
		} else {
			wantDesc = c.input.Error()
		}

		if got.Description != wantDesc {
			t.Errorf("ErrorAck(%q).Description = %s; want %s", c.input, got.Description, wantDesc)
		}
	}
}
