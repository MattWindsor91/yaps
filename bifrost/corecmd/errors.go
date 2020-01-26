package corecmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/UniversityRadioYork/baps3d/bifrost/msgproto"
)

// File corecmd/errors contains structured errors for core request/response parsers.

// WordError is sent when a parser expects one word, but parses another.
type WordError struct {
	// Got is the word that the parser got.
	Got string

	// Want is the word that the parser expected.
	Want string
}

func (w WordError) Error() string {
	return fmt.Sprintf("message word is '%s', want '%s'", w.Got, w.Want)
}

func (w WordError) Blame() Blame {
	return BlameClient
}

// CheckWord checks to see if the message m has the command words want, up to any whitespace.
// It returns a WordError if not.
// CheckWord is case sensitive, because uppercase and lowercase words have different meanings.
func CheckWord(want string, m *msgproto.Message) error {
	got := m.Word()
	if want != strings.TrimSpace(got) {
		return WordError{Got: got, Want: want}
	}
	return nil
}

// ArityError is sent when a parser expects a certain number of arguments, but gets a wrong amount.
type ArityError struct {
	// Got is the number of arguments the parser got.
	Got int

	// Min is the minimum number of arguments the parser expected.
	Min int

	// Max is the maximum number of arguments the parser expected.
	Max int
}

func (a ArityError) Error() string {
	return fmt.Sprintf("message has %s, want %s", a.got(), a.want())
}

func (a ArityError) got() string {
	if a.Got == 1 {
		return "one argument"
	}
	return fmt.Sprintf("%d arguments", a.Got)
}

func (a ArityError) want() string {
	if a.Min != a.Max {
		return fmt.Sprintf("%d-%d", a.Min, a.Max)
	}
	return strconv.Itoa(a.Min)
}

func (a ArityError) Blame() Blame {
	return BlameClient
}

// CheckArity checks to see if the number of arguments in message m is between min and max inclusive.
// It returns the arguments if so, and an ArityError if not.
func CheckArity(min, max int, m *msgproto.Message) (got []string, err error) {
	got = m.Args()
	l := len(got)
	if l < min || max < l {
		err = ArityError{Got: l, Min: min, Max: max}
	}
	return got, err
}

// OneArg checks to see if m has one argument precisely.
// If so, OneArg returns it.
// If not, OneArg returns an ArityError.
func OneArg(m *msgproto.Message) (arg string, err error) {
	var got []string
	if got, err = CheckArity(1, 1, m); err != nil {
		return "", err
	}
	return got[0], nil
}

// TwoArgs checks to see if m has two arguments precisely.
// If so, TwoArgs returns them.
// If not, TwoArgs returns an ArityError.
func TwoArgs(m *msgproto.Message) (arg1, arg2 string, err error) {
	var got []string
	if got, err = CheckArity(2, 2, m); err != nil {
		return "", "", err
	}
	return got[0], got[1], nil
}
