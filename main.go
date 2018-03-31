package main

import (
	"fmt"
	"log"
	"os"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/UniversityRadioYork/baps3d/comm"
	"github.com/UniversityRadioYork/baps3d/console"
	"github.com/UniversityRadioYork/baps3d/list"
	"github.com/UniversityRadioYork/baps3d/netsrv"	
)

func copyClient(cli *comm.Client) *comm.Client {
	sdreply := make(chan comm.Response)
	cli.Tx <- comm.Request{
		Origin: comm.RequestOrigin{
			Tag:     bifrost.TagUnknown,
			ReplyTx: sdreply,
		},
		Body: comm.NewClientRequest{},
	}
	var ncli *comm.Client
	for {
		r := <-sdreply
		switch b := r.Body.(type) {
		case comm.NewClientResponse:
			ncli = b.Client
		case comm.AckResponse:
			return ncli
		}
	}
}

func main() {
	lst := list.New()
	lstCon, rootClient := comm.NewController(lst)
	go lstCon.Run()

	netLog := log.New(os.Stderr, "net", log.LstdFlags)
	netClient := copyClient(rootClient)
	netSrv := netsrv.New(netLog, "localhost:1357", netClient, lst)
	go netSrv.Run()
	
	consoleLstClient := copyClient(rootClient)
	consoleBf, consoleBfClient := comm.NewBifrost(consoleLstClient, lst)
	go consoleBf.Run()
	console, err := console.New(consoleBfClient)
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
	rootClient.Shutdown()
	fmt.Println("got shutdown request ack")
	for range consoleBfClient.Rx {
	}
}
