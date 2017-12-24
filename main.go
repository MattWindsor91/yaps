package main

import (
	"fmt"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/UniversityRadioYork/baps3d/comm"
	"github.com/UniversityRadioYork/baps3d/console"
	"github.com/UniversityRadioYork/baps3d/list"
)

func main() {
	lc, cli := list.NewControlledList()
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
	if err = console.Close(); err != nil {
		fmt.Println(err)
	}
	fmt.Println("shutting down")
	sdreply := make(chan comm.Response)
	cli.Tx <- comm.Request{
		Origin: comm.RequestOrigin{
			Tag:     bifrost.TagUnknown,
			ReplyTx: sdreply,
		},
		Body: comm.ShutdownRequest{},
	}
	fmt.Println("sent shutdown request")
	<-sdreply
	fmt.Println("got shutdown request ack")
	for range lrx {
	}
}
