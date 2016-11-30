package main

import "github.com/UniversityRadioYork/baps3d/list"

func spinUpList() *list.Controller {
	lst := list.New()
	return list.NewController(lst)
}

func main() {
	lc := spinUpList()
	go lc.Run()
}
