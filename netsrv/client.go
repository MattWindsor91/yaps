package netsrv

import (
	"io"
	"log"
	"sync"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/UniversityRadioYork/baps3d/comm"
)

// Client holds the server-side state of a baps3d Bifrost client.
type Client struct {
	// name holds a descriptive name for the Client.
	name string

	// log holds the logger for this client.
	log *log.Logger

	// conn holds the internal client.
	conn io.ReadWriteCloser

	// conClient is the client's Client for the Controller for this
	// server.
	conClient *comm.Client

	// conBifrost is the Bifrost adapter for conClient.
	conBifrost *comm.BifrostClient
}

// Close closes the given client.
func (c *Client) Close() error {
	// TODO(@MattWindsor91): disconnect client and bifrost
	return c.conn.Close()
}

// Run spins up the client's receiver and transmitter loops.
// It takes the client's Bifrost adapter,
// and the server's client hangup and done channels.
func (c *Client) Run(bifrost *comm.Bifrost, hangUp chan<- *Client, done <-chan struct{}) {
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		c.runTx()
		// Only hang up if the server is still around.
		// Otherwise, we'll just hang here waiting for the server to answer,
		// while the server hangs up the client anyway.
		select {
		case hangUp <- c:
		case <-done:
		}
		wg.Done()
	}()

	go func() {
		c.runRx()
		wg.Done()
	}()

	go func() {
		bifrost.Run()
		wg.Done()
	}()

	wg.Wait()
}

// runRx runs the client's message receiver loop.
// This writes messages to the socket.
func (c *Client) runRx() {
	// We don't have to check c.bclient.Done here:
	// client always drops both Rx and Done when shutting down.
	for m := range c.conBifrost.Rx {
		mbytes, err := m.Pack()
		if err != nil {
			c.outputError(err)
			continue
		}

		if _, err := c.conn.Write(mbytes); err != nil {
			c.outputError(err)
			break
		}
	}
}

// outputError logs a connection error for client c.
func (c *Client) outputError(e error) {
	c.log.Printf("connection error on %s: %s\n", c.name, e.Error())
}

// runTx runs the client's message transmitter loop.
// This reads from stdin.
func (c *Client) runTx() {
	r := bifrost.NewReaderTokeniser(c.conn)

	for {
		line, terr := r.ReadLine()
		if terr != nil {
			c.outputError(terr)
			break
		}

		msg, merr := bifrost.LineToMessage(line)
		if merr != nil {
			c.outputError(merr)
			break
		}

		if !c.conBifrost.Send(*msg) {
			c.log.Printf("client died while sending message on %s", c.name)
			break
		}
	}
}
