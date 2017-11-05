package list_test

import (
	"fmt"
	"testing"

	"github.com/UniversityRadioYork/baps3d/list"
)

func ExampleAutoMode_String() {
	fmt.Println(list.AutoOff.String())

	// Output:
	// off
}

// TestAutoModeString tests the String method of AutoMode.
func TestAutoModeString(t *testing.T) {
	cases := []struct {
		a list.AutoMode
		s string
	}{
		{list.AutoOff, "off"},
		{list.AutoDrop, "drop"},
		{list.AutoNext, "next"},
		{list.AutoShuffle, "shuffle"},
		{list.AutoShuffle + 1, "?unknown?"},
	}

	for _, c := range cases {
		g := c.a.String()
		if g != c.s {
			t.Fatalf("%v.String() was '%s', should be '%s'", c.a, g, c.s)
		}
	}
}

func ExampleParseAutoMode() {
	a, e := list.ParseAutoMode("off")
	fmt.Println(a)
	fmt.Println(e)

	// Output:
	// off
	// <nil>
}

// TestAutoModeParseIdempotence checks that parsing the string version of an AutoMode is the identity.
func TestAutoModeParseIdempotence(t *testing.T) {
	for i := list.FirstAuto; i <= list.LastAuto; i++ {
		a, e := list.ParseAutoMode(i.String())
		if e != nil {
			t.Errorf("unexpected parse error: %A", e)
		} else if a != i {
			t.Errorf("%A parsed as %A", i, a)
		}
	}
}
