package list_test

import (
	"fmt"

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
