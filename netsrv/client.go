package netsrv

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/UniversityRadioYork/baps3d/controller"
)

// Client holds the server-side state of a baps3d Bifrost client.
type Client struct {
	// name holds a descriptive name for the Client.
	name string

	// log holds the logger for this client.
	log *log.Logger

	// conClient is the client's Client for the Controller for this
	// server.
	conClient *controller.Client

	// ioClient is the underlying Bifrost-level client.
	ioClient *bifrost.IoClient
}

// Close closes the given client.
func (c *Client) Close() error {
	// TODO(@MattWindsor91): disconnect client and bifrost
	return c.ioClient.Close()
}

// Run spins up the client's receiver and transmitter loops.
// It takes the server context, the client's Bifrost adapter, and the server's client hangup channel.
func (c *Client) Run(ctx context.Context, bf *controller.Bifrost, hangUp chan<- *Client) {
	var wg sync.WaitGroup
	wg.Add(3)

	errCh := make(chan error)

	go func() {
		c.ioClient.Run(ctx, errCh)
		wg.Done()
	}()

	go func() {
		c.handleIoErrors(errCh, hangUp)
		wg.Done()
	}()

	go func() {
		bf.Run(ctx)
		wg.Done()
	}()

	wg.Wait()
}

// handleIoErrors monitors errCh for errors, forwarding any hangup requests coming through to hangUp and logging all
// other errors.
func (c *Client) handleIoErrors(errCh <-chan error, hangUp chan<- *Client) {
	for err := range errCh {
		if errors.Is(err, bifrost.HungUpError) {
			hangUp <- c
		} else {
			c.outputError(err)
		}
	}
}

// outputError logs a connection error for client c.
func (c *Client) outputError(e error) {
	c.log.Printf("connection error on %s: %s\n", c.name, e.Error())
}
