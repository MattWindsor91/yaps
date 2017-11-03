package list_test

import (
	"fmt"
	"testing"

	"github.com/UniversityRadioYork/baps3d/list"
)

func ExampleNew() {
	l := list.New()

	fmt.Println(l.Count())

	idx, _ := l.Selection()
	fmt.Println(idx)

	fmt.Println(l.AutoMode())

	// Output:
	// 0
	// -1
	// off
}

// ExampleList_SetAutoMode tests List.SetAutoMode in an example style.
func ExampleList_SetAutoMode() {
	l := list.New()

	l.SetAutoMode(list.AutoOff)
	fmt.Println(l.AutoMode())

	l.SetAutoMode(list.AutoDrop)
	fmt.Println(l.AutoMode())

	l.SetAutoMode(list.AutoNext)
	fmt.Println(l.AutoMode())

	l.SetAutoMode(list.AutoShuffle)
	fmt.Println(l.AutoMode())

	// Output:
	// off
	// drop
	// next
	// shuffle
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

// ExampleList_Selection tests List.Selection in an example style.
func ExampleList_Selection() {
	// New lists have no selection.
	l := list.New()

	idx, _ := l.Selection()
	fmt.Println(idx)

	// If we change the selection, Selection updates.
	if err := l.Add(list.NewTrack("xyz", "foo.mp3"), 0); err != nil {
		panic(err)
	}
	if _, err := l.Select(0, "xyz"); err != nil {
		panic(err)
	}

	idx, _ = l.Selection()
	fmt.Println(idx)

	// Output:
	// -1
	// 0
}
