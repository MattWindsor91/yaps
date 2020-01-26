package corecmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/UniversityRadioYork/baps3d/bifrost/msgproto"
)

// ExampleOneArg is a testable example for OneArg.
func ExampleOneArg() {
	m := msgproto.NewMessage("foo", "bar").AddArgs("baz")
	if arg, err := OneArg(m); err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Println(arg)
	}

	// Output:
	// baz
}

// ExampleTwoArgs is a testable example for TwoArgs.
func ExampleTwoArgs() {
	m := msgproto.NewMessage("foo", "bar").AddArgs("baz", "barbaz")
	if arg1, arg2, err := TwoArgs(m); err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Println(arg1)
		fmt.Println(arg2)
	}

	// Output:
	// baz
	// barbaz
}

var oneArgErrorCases = [][]string{
	{},
	{"foo", "bar"},
}

var twoArgsErrorCases = [][]string{
	{},
	{"foo"},
	{"foo", "bar", "baz"},
}

var arityErrorCases = []struct {
	in  ArityError
	out string
}{
	{in: ArityError{Got: 1, Min: 0, Max: 0},
		out: "message has one argument, want 0",
	},
	{in: ArityError{Got: 0, Min: 1, Max: 2},
		out: "message has 0 arguments, want 1-2",
	},
	{in: ArityError{Got: 2, Min: 0, Max: 1},
		out: "message has 2 arguments, want 0-1",
	},
	{in: ArityError{Got: 3, Min: 1, Max: 1},
		out: "message has 3 arguments, want 1",
	},
}

// TestArityError_Error tests the Error output for a variety of ArityErrors.
func TestArityError_Error(t *testing.T) {
	for _, c := range arityErrorCases {
		if got := c.in.Error(); got != c.out {
			t.Errorf("(%v).Error()=%q; want %q", c.in, got, c.out)
		}
	}
}

// TestOneArg_error exercises OneArg's error handling.
func TestOneArg_error(t *testing.T) {
	for _, args := range oneArgErrorCases {
		m := msgproto.NewMessage(msgproto.TagBcast, "YEET").AddArgs(args...)

		if arg, err := OneArg(m); arg != "" {
			t.Errorf("non-empty return from bad OneArg: %q", arg)
		} else {
			testArgUnpackError(t, "OneArg", 1, len(args), err)
		}
	}
}

// TestTwoArgs_error exercises TwoArgs's error handling.
func TestTwoArgs_error(t *testing.T) {
	for _, args := range twoArgsErrorCases {
		m := msgproto.NewMessage(msgproto.TagBcast, "YEET").AddArgs(args...)

		if arg1, arg2, err := TwoArgs(m); arg1 != "" {
			t.Errorf("non-empty first return from bad TwoArgs: %q", arg1)
		} else if arg2 != "" {
			t.Errorf("non-empty second return from bad TwoArgs: %q", arg2)
		} else {
			testArgUnpackError(t, "TwoArgs", 2, len(args), err)
		}
	}
}

func testArgUnpackError(t *testing.T, name string, wantCount, gotCount int, err error) {
	t.Helper()

	var aerr ArityError
	if err == nil {
		t.Errorf("%s error nil; want ArityError", name)
	} else if !errors.As(err, &aerr) {
		t.Errorf("%s error not ArityError: %v", name, err)
	} else if aerr.Max != wantCount {
		t.Errorf("%s error has weird Max=%d", name, aerr.Max)
	} else if aerr.Min != wantCount {
		t.Errorf("%s error has weird Min=%d", name, aerr.Min)
	} else if aerr.Got != gotCount {
		t.Errorf("%s error has Got=%d; want %d", name, aerr.Got, gotCount)
	}
}

func testParserWordError(t *testing.T, err error, want, got string) {
	t.Helper()

	var w WordError
	if err == nil {
		t.Errorf("no error when parsing %s bad-word %s", want, got)
	} else if !errors.As(err, &w) {
		t.Errorf("non-WordError when parsing %s bad-word %s: %v", want, got, err)
	} else if w.Want != want {
		t.Errorf("ack WordError has want=%q; should be %q", w.Want, want)
	} else if w.Got != got {
		t.Errorf("ack ArityError has got=%q; should be %q", w.Got, got)
	}
}
