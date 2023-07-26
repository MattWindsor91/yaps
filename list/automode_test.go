package list_test

import (
	"fmt"
	"testing"

	"github.com/MattWindsor91/yaps/list"
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

// TestParseAutoMode_Valid tests ParseAutoMode with valid strings.
func TestParseAutoMode_Valid(t *testing.T) {
	cases := []struct {
		a list.AutoMode
		s string
	}{
		{list.AutoOff, "off"},
		{list.AutoDrop, "drop"},
		{list.AutoNext, "next"},
		{list.AutoShuffle, "shuffle"},
	}

	for _, c := range cases {
		g, e := list.ParseAutoMode(c.s)
		if e != nil {
			t.Fatalf("unexpected error: %s", e.Error())
		}
		if g != c.a {
			t.Fatalf("'%s' parsed as '%s', not %v", c.s, g, c.a)
		}
	}
}

// TestParseAutoMode_Invalid tests ParseAutoMode with invalid strings.
func TestParseAutoMode_Invalid(t *testing.T) {
	cases := []string{
		"",
		" ",
		"\n",
		" off",
		"drop ",
		" next ",
		"shuffle\n",
		"invalid",
	}

	for _, c := range cases {
		g, e := list.ParseAutoMode(c)
		if e == nil {
			t.Fatalf("invalid automode '%s' parsed as %v", c, g)
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
			t.Errorf("unexpected parse error: %v", e)
		} else if a != i {
			t.Errorf("%v parsed as %v", i, a)
		}
	}
}
