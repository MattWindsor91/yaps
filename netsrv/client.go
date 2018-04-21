package netsrv

import (
	"log"
	"net"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/UniversityRadioYork/baps3d/comm"
)

// client holds the server-side state of a baps3d TCP client.
type client struct {
	// log holds the logger for this client.
	log *log.Logger

	// conn holds the client socket.
	conn net.Conn

	// buf holds the client buffer.
	buf [4096]byte

	// conClient is the client's Client for the Controller for this
	// server.
	conClient *comm.Client

	// conBifrost is the Bifrost adapter for conClient.
	conBifrost *comm.BifrostClient
}

// Close closes the given client.
func (c *client) Close() error {
	// TODO(@MattWindsor91): disconnect client and bifrost
	return c.conn.Close()
}

// RunRx runs the client's message receiver loop.
// This writes messages to the socket.
func (c *client) RunRx() {
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
func (c *client) outputError(e error) {
	c.log.Println("connection error:", e.Error())
}

// RunTx runs the client's message transmitter loop.
// This reads from stdin.
func (c *client) RunTx() {
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
			c.log.Println("client died while sending message")
			break
		}
	}
}
