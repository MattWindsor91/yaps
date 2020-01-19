package bifrost

import (
	"context"
	"errors"
	"github.com/UniversityRadioYork/baps3d/bifrost/msgproto"
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
	Bifrost *Endpoint
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
func (c *IoClient) Run(ctx context.Context, errCh chan<- error) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		c.runTx(ctx, errCh)
		c.sendError(ctx, errCh, HungUpError)
		wg.Done()
	}()

	go func() {
		c.runRx(ctx, errCh)
		wg.Done()
	}()

	wg.Wait()
	close(errCh)
}

// runRx runs the client's message receiver loop.
// This writes messages to the socket.
func (c *IoClient) runRx(ctx context.Context, errCh chan<- error) {
	// We don't have to check c.Bifrost.Done here:
	// client always drops both Rx and Done when shutting down.
	for m := range c.Bifrost.Rx {
		mbytes, err := m.Pack()
		if err != nil {
			c.sendError(ctx, errCh, err)
			continue
		}

		if _, err := c.Conn.Write(mbytes); err != nil {
			c.sendError(ctx, errCh, err)
			break
		}
	}
}

// runTx runs the client's message transmitter loop.
func (c *IoClient) runTx(ctx context.Context, errCh chan<- error) {
	r := msgproto.NewReaderTokeniser(c.Conn)

	for {
		if err := c.txLine(ctx, r); err != nil {
			c.sendError(ctx, errCh, err)
			return
		}
	}
}

// txLine transmits a line from the ReaderTokeniser r
func (c *IoClient) txLine(ctx context.Context, r *msgproto.ReaderTokeniser) (err error) {
	var line []string
	if line, err = r.ReadLine(); err != nil {
		return err
	}

	var msg *msgproto.Message
	if msg, err = msgproto.LineToMessage(line); err != nil {
		return err
	}

	if !c.Bifrost.Send(ctx, *msg) {
		return errors.New("client died while sending message on %s")
	}

	return nil
}

// sendError tries to send an error e to the error channel errCh.
// It silently fails if the underlying Client's Done channel is closed.
func (c *IoClient) sendError(ctx context.Context, errCh chan<- error, e error) {
	done := ctx.Done()
	select {
	case errCh <- e:
	case <-done:
	}
}
