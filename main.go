package main

import (
	"fmt"

	"github.com/UniversityRadioYork/baps3d/bifrost"
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
	console, err := console.New(ltx, lrx)
	if err != nil {
		fmt.Println(err)
		return
	}

	go console.RunRx()
	console.RunTx()
	console.Close()
	fmt.Println("shutting down")
	sdreply := make(chan list.Response)
	cli.Tx <- list.Request{
		Origin: list.RequestOrigin{
			Tag:     bifrost.TagUnknown,
			ReplyTx: sdreply,
		},
		Body: list.ShutdownRequest{},
	}
	fmt.Println("sent shutdown request")
	_ = <-sdreply
	fmt.Println("got shutdown request ack")
	for _ = range lrx {
	}
}
