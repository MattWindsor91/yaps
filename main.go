package main

import (
	"fmt"
	"log"
	"os"

	"github.com/UniversityRadioYork/baps3d/comm"
	"github.com/UniversityRadioYork/baps3d/console"
	"github.com/UniversityRadioYork/baps3d/list"
	"github.com/UniversityRadioYork/baps3d/netsrv"	
)

func main() {
	rootLog := log.New(os.Stderr, "[root] ", log.LstdFlags)
	
	lst := list.New()
	lstCon, rootClient := comm.NewController(lst)
	go lstCon.Run()

	netLog := log.New(os.Stderr, "[net] ", log.LstdFlags)
	netClient, err := rootClient.Copy()
	if err != nil {
		rootLog.Println("couldn't create network client:", err)
		return
	}
	netSrv := netsrv.New(netLog, "localhost:1357", netClient, lst)
	go netSrv.Run()
	
	consoleLstClient, err := rootClient.Copy()
	if err != nil {
		rootLog.Println("couldn't create console client:", err)
		return
	}
	consoleBf, consoleBfClient := comm.NewBifrost(consoleLstClient, lst)
	go consoleBf.Run()
	console, err := console.New(consoleBfClient)
	if err != nil {
		rootLog.Println("couldn't bring up console:", err)
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
