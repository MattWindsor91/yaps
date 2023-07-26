package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"

	"github.com/MattWindsor91/yaps/config"
	"golang.org/x/sync/errgroup"

	"github.com/MattWindsor91/yaps/console"
	"github.com/MattWindsor91/yaps/controller"
	"github.com/MattWindsor91/yaps/list"
	"github.com/MattWindsor91/yaps/netsrv"
)

func makeLog(section string, enabled bool) *log.Logger {
	var lw io.Writer
	if enabled {
		lw = os.Stderr
	} else {
		lw = io.Discard
	}

	return log.New(lw, "["+section+"] ", log.LstdFlags)
}

func runNet(ctx context.Context, rootClient *controller.Client, ncfg config.Net) error {
	netClient, err := rootClient.Copy(ctx)
	if err != nil {
		return err
	}

	netLog := makeLog("net", ncfg.Log)
	netSrv := netsrv.New(netLog, ncfg.Host, netClient)
	netSrv.Run(ctx)
	return nil
}

func runConsole(ctx context.Context, rootClient *controller.Client, ccfg config.Console) error {
	consoleClient, err := rootClient.Copy(ctx)
	if err != nil {
		return err
	}

	con, err := console.New(ctx, consoleClient)
	if err != nil {
		return err
	}
	return con.Run(ctx)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	rootLog := makeLog("root", true)

	cfile := "yaps.toml"
	conf, err := config.Parse(cfile)
	if err != nil {
		rootLog.Printf("couldn't open config: %v\n", err)
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	var errg errgroup.Group

	if len(conf.Lists) != 1 {
		rootLog.Printf("FIXME: must have precisely one configured list, got %d\n", len(conf.Lists))
		return
	}
	// lstConf := conf.Lists[0]

	lst := list.New()
	lstCon, rootClient := controller.NewController(lst)
	errg.Go(func() error {
		lstCon.Run(ctx)
		rootLog.Println("list controller closing")
		return nil
	})

	if conf.Net.Enabled {
		errg.Go(func() error {
			err := runNet(ctx, rootClient, conf.Net)
			if err != nil {
				err = fmt.Errorf("netsrv error: %w", err)
			}
			rootLog.Println("netsrv closing")
			return err
		})
	}

	if conf.Console.Enabled {
		errg.Go(func() error {
			err := runConsole(ctx, rootClient, conf.Console)
			if err != nil {
				err = fmt.Errorf("console error: %w", err)
			}
			rootLog.Println("console closing")
			return err
		})
	}

	mainLoop(rootClient, interrupt, ctx, rootLog)
	cancel()

	rootLog.Println("Waiting for subsystems to shut down...")
	if err := errg.Wait(); err != nil {
		rootLog.Printf("main subsystem error: %s", err.Error())
	}
	rootLog.Println("It's now safe to turn off your yaps.")
}

func mainLoop(rootClient *controller.Client, interrupt chan os.Signal, ctx context.Context, rootLog *log.Logger) {
	running := true
	for running {
		select {
		case _, running = <-rootClient.Rx:
			// Accept, but ignore, all messages from the root client.
			// Start closing yaps if the client has closed.
		case <-interrupt:
			// Ctrl-C, so gracefully shut down.
			if err := rootClient.Shutdown(ctx); err != nil {
				rootLog.Println("couldn't shut down gracefully:", err)
			}
		}
	}
}
