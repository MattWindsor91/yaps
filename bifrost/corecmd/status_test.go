package corecmd

import (
	"errors"
	"fmt"
	"testing"
)

// TestStatus_String tests whether statuses convert to the appropriate strings.
func TestStatus_String(t *testing.T) {
	cases := []struct {
		input Status
		want  string
	}{
		{StatusOk, WordOk},
		{StatusWhat, WordWhat},
		{StatusFail, WordFail},
		{StatusUnknown, "unknown Status: 0"},
	}

	for _, c := range cases {
		got := c.input.String()
		if got != c.want {
			t.Errorf("(%q).String() = '%s'; want '%s'", c.input, got, c.want)
		}
	}
}

// TestParseStatus_RoundTrip checks that parsing the String of a Status returns the same Status.
func TestParseStatus_RoundTrip(t *testing.T) {
	statuses := []Status{StatusOk, StatusWhat, StatusFail}

	for _, s := range statuses {
		str := s.String()
		if got, err := ParseStatus(str); err != nil {
			t.Errorf("round-trip on %q (%s) gave error %s", s, str, err.Error())
		} else if got != s {
			t.Errorf("round-trip on %q (%s) produced %q (%s)", s, str, got, got.String())
		}
	}
}

// KnownBadStatus is a word that is known to trigger a status parsing error.
const KnownBadStatus = "eef freef!"

// TestParseStatus_error checks that parsing a bad Status raises an appropriate error.
func TestParseStatus_error(t *testing.T) {
	s, err := ParseStatus(KnownBadStatus)
	if s != StatusUnknown {
		t.Errorf("parse of bad status = %s", s)
	}
	if err == nil {
		t.Fatal("parse of bad status gave no error")
	}

	checkStatusError(t, err, KnownBadStatus)
}

// checkStatusError is the common testing glue for Status and AckResponse related status error parsing.
func checkStatusError(t *testing.T, err error, input string) {
	t.Helper()

	if b := ErrorBlame(err); b != BlameClient {
		t.Errorf("parse of bad status blame=%q, not client", b)
	}

	got := err.Error()
	want := fmt.Sprintf("bad status: %q", input)
	if got != want {
		t.Errorf("parse of bad status gave error %q; want %q", got, want)
	}
}

type failError struct{}

func (f failError) Error() string {
	return "f"
}
func (f failError) Blame() Blame {
	return BlameServer
}

// TestErrorStatus tests ErrorStatus with various error inputs.
func TestErrorStatus(t *testing.T) {
	cases := []struct {
		input error
		want  Status
	}{
		{WordError{
			Got:  "ack",
			Want: "ACK",
		}, StatusWhat},
		{failError{}, StatusFail},
		{errors.New("anonymous"), StatusFail},
	}

	for _, c := range cases {
		if got := ErrorStatus(c.input); got != c.want {
			t.Errorf("ErrorStatus(%q) = %q; want %q", c.input, got, c.want)
		}
	}
}
