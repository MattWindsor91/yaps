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


func ExampleList_SetAutoMode() {
	l := list.New()
	fmt.Println(l.AutoMode())

	l.SetAutoMode(list.AutoShuffle)
	fmt.Println(l.AutoMode())

	// Output:
	// off
	// shuffle
}


func ExampleList_Selection() {
	// New lists have no selection.
	l := list.New()

	idx, _ := l.Selection()
	fmt.Println(idx)

	// Output:
	// -1
}
