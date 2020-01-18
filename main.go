package main

import (
	"github.com/UniversityRadioYork/baps3d/config"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/UniversityRadioYork/baps3d/comm"
	"github.com/UniversityRadioYork/baps3d/console"
	"github.com/UniversityRadioYork/baps3d/list"
	"github.com/UniversityRadioYork/baps3d/netsrv"
)

func makeLog(section string, enabled bool) *log.Logger {
	var lw io.Writer
	if enabled {
		lw = os.Stderr
	} else {
		lw = ioutil.Discard
	}

	return log.New(lw, "["+section+"] ", log.LstdFlags)
}

func runNet(rootClient *comm.Client, ncfg config.Net) error {
	netClient, err := rootClient.Copy()
	if err != nil {
		return err
	}

	netLog := makeLog("net", ncfg.Log)
	netSrv := netsrv.New(netLog, ncfg.Host, netClient)
	netSrv.Run()
	return nil
}

func runConsole(rootClient *comm.Client, ccfg config.Console) error {
	consoleClient, err := rootClient.Copy()
	if err != nil {
		return err
	}

	console, err := console.New(consoleClient)
	if err != nil {
		return err
	}
	return console.Run()
}

func main() {
	rootLog := makeLog("root", true)

	cfile := "baps3d.toml"
	conf, err := config.Parse(cfile)
	if err != nil {
		rootLog.Printf("couldn't open config: %s\n", err.Error())
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	var wg sync.WaitGroup

	lst := list.New()
	lstCon, rootClient := comm.NewController(lst)
	wg.Add(1)
	go func() {
		lstCon.Run()
		wg.Done()
	}()

	if conf.Net.Enabled {
		wg.Add(1)
		go func() {
			if err := runNet(rootClient, conf.Net); err != nil {
				rootLog.Println("netsrv error:", err)
			}
			wg.Done()
		}()
	}

	if conf.Console.Enabled {
		wg.Add(1)
		go func() {
			if err := runConsole(rootClient, conf.Console); err != nil {
				rootLog.Println("console error:", err)
			}
			wg.Done()
		}()
	}

	running := true
	for running {
		select {
		case _, running = <-rootClient.Rx:
			// Accept, but ignore, all messages from the root client.
			// Start closing baps3d if the client has closed.
		case <-interrupt:
			// Ctrl-C, so gracefully shut down.
			if err := rootClient.Shutdown(); err != nil {
				rootLog.Println("couldn't shut down gracefully:", err)
			}
		}
	}

	rootLog.Println("Waiting for subsystems to shut down...")
	wg.Wait()
	rootLog.Println("It's now safe to turn off your baps3d.")
}
