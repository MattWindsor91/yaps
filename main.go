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

	lb, ltx, lrx := list.NewBifrost(cli)
	go lb.Run()
	console := console.New(ltx, lrx)

	go console.RunRx()
	console.RunTx()
}
