package main

import (
	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/UniversityRadioYork/baps3d/console"
	"github.com/UniversityRadioYork/baps3d/list"
)

func spinUpList() (*list.Controller, *list.Client) {
	lst := list.New()
	return list.NewController(lst)
}

func main() {
	lc, _ := spinUpList()
	go lc.Run()

	dummy := make(chan bifrost.Message)
	console := console.New(dummy)

	go console.RunRx()
	console.RunTx()
}
