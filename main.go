package main

import (
	"fmt"
	"log"
	"os"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/UniversityRadioYork/baps3d/comm"
	"github.com/UniversityRadioYork/baps3d/console"
	"github.com/UniversityRadioYork/baps3d/list"
	"github.com/UniversityRadioYork/baps3d/netclient"	
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
	lc, cli := list.NewControlledList()
	go lc.Run()
	lb, lbc := list.NewBifrost(cli)
	go lb.Run()

	netLog := log.New(os.Stderr, "net", log.LstdFlags)
	netClient := copyClient(cli)
	netBifrost, _ := lb.Fork(netClient)
	netSrv := netclient.NewServer(netLog, "localhost:1357", netClient, netBifrost)
	go netSrv.Run()
	
	console, err := console.New(lbc)
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
	cli.Shutdown()
	fmt.Println("got shutdown request ack")
	for range lbc.Rx {
	}
}
