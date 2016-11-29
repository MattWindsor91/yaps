package list_test

import (
	"fmt"

	"github.com/UniversityRadioYork/baps3d/list"
)

func ExampleList_SetAutoMode() {
	l := list.New()
	fmt.Println(l.GetAutoMode().String())

	l.SetAutoMode(list.AutoShuffle)
	fmt.Println(l.GetAutoMode().String())
	// Output:
	// off
	// shuffle
}
