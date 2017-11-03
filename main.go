package main

import (
	"github.com/UniversityRadioYork/baps3d/console"
	"github.com/UniversityRadioYork/baps3d/list"
)

func spinUpList() (*list.Controller, *list.Client) {
	lst := list.New()
	return list.NewController(lst)
}

func main() {
	lc, cli := spinUpList()
	go lc.Run()

	lb, lmsgs := list.NewBifrost(cli)
	go lb.Run()
	console := console.New(lmsgs)

	go console.RunRx()
	console.RunTx()
}
