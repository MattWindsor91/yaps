package bifrost

import (
	"errors"
	"io"
	"sync"
)

// HungUpError is the error sent by an IoClient when its transmission loop has hung up.
var HungUpError = errors.New("client has hung up")

// IoClient represents a Bifrost client that sends and receives messages along an I/O connection.
type IoClient struct {
	// conn holds the internal I/O connection.
	Conn io.ReadWriteCloser

	// bifrost holds the Bifrost channel pair used by the IoClient.
	Bifrost *Client
}

func (c *IoClient) Close() error {
	// TODO(@MattWindsor91): make sure we close everything
	close(c.Bifrost.Tx)
	return c.Conn.Close()
}

// Run spins up the client's receiver and transmitter loops.
// It takes a channel to notify the caller asynchronously of any errors, and a client
// and the server's client hangup and done channels.
// It closes errors once both loops are done.
func (c *IoClient) Run(errCh chan<- error) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		c.runTx(errCh)
		c.sendError(errCh, HungUpError)
		wg.Done()
	}()

	go func() {
		c.runRx(errCh)
		wg.Done()
	}()

	wg.Wait()
	close(errCh)
}

// runRx runs the client's message receiver loop.
// This writes messages to the socket.
func (c *IoClient) runRx(errCh chan<- error) {
	// We don't have to check c.Bifrost.Done here:
	// client always drops both Rx and Done when shutting down.
	for m := range c.Bifrost.Rx {
		mbytes, err := m.Pack()
		if err != nil {
			c.sendError(errCh, err)
			continue
		}

		if _, err := c.Conn.Write(mbytes); err != nil {
			c.sendError(errCh, err)
			break
		}
	}
}

// runTx runs the client's message transmitter loop.
func (c *IoClient) runTx(errCh chan<- error) {
	r := NewReaderTokeniser(c.Conn)

	for {
		if err := c.txLine(r); err != nil {
			c.sendError(errCh, err)
			return
		}
	}
}

// txLine transmits a line from the ReaderTokeniser r
func (c *IoClient) txLine(r *ReaderTokeniser) (err error) {
	var line []string
	if line, err = r.ReadLine(); err != nil {
		return err
	}

	var msg *Message
	if msg, err = LineToMessage(line); err != nil {
		return err
	}

	if !c.Bifrost.Send(*msg) {
		return errors.New("client died while sending message on %s")
	}

	return nil
}

// sendError tries to send an error e to the error channel errCh.
// It silently fails if the underlying Client's Done channel is closed.
func (c *IoClient) sendError(errCh chan<- error, e error) {
	select {
	case errCh<- e:
	case <-c.Bifrost.Done:
	}
}


