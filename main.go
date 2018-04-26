package main

import (
	"log"
	"os"
	"sync"

	"github.com/UniversityRadioYork/baps3d/comm"
	"github.com/UniversityRadioYork/baps3d/console"
	"github.com/UniversityRadioYork/baps3d/list"
	"github.com/UniversityRadioYork/baps3d/netsrv"
)

func main() {
	var wg sync.WaitGroup

	rootLog := log.New(os.Stderr, "[root] ", log.LstdFlags)

	lst := list.New()
	lstCon, rootClient := comm.NewController(lst)
	wg.Add(1)
	go func() {
		lstCon.Run()
		wg.Done()
	}()

	netLog := log.New(os.Stderr, "[net] ", log.LstdFlags)
	netClient, err := rootClient.Copy()
	if err != nil {
		rootLog.Println("couldn't create network client:", err)
		return
	}
	netSrv := netsrv.New(netLog, "localhost:1357", netClient)
	wg.Add(1)
	go func() {
		netSrv.Run()
		wg.Done()
	}()

	consoleClient, err := rootClient.Copy()
	if err != nil {
		rootLog.Println("couldn't create console client:", err)
		return
	}
	console, err := console.New(consoleClient)
	if err != nil {
		rootLog.Println("couldn't bring up console:", err)
		return
	}

	wg.Add(1)
	go func() {
		if err := console.Run(); err != nil {
			rootLog.Println("error closing console:", err)
		}
		consoleClient.Shutdown()
		wg.Done()
	}()

	for range rootClient.Rx {
	}
	wg.Wait()
	rootLog.Println("It's now safe to turn off your baps3d.")
}
