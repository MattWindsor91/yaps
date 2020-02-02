// Package console is a simple console for inputting Bifrost commands.
package console

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/UniversityRadioYork/baps3d/bifrost/msgproto"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/UniversityRadioYork/baps3d/controller"
	"github.com/chzyer/readline"
)

const (
	// Console request prompts
	// (Must include trailing space)
	promptNormal   = "$ "
	promptContinue = "> "
	// Console response prefixes
	// (Must _not_ include trailing space)
	prefixMessage = "[R]"
	prefixError   = "[!]"
)

// Console provides a readline-style console for sending Bifrost messages to a controller.
type Console struct {
	client  *controller.Client
	bf      *controller.Bifrost
	bclient *bifrost.Endpoint
	tok     *msgproto.Tokeniser
	rl      *readline.Instance
	txrun   bool
}

// New creates a new Console.
// This can fail if the underlying console library fails, or if the Client
// doesn't support Bifrost.
func New(ctx context.Context, client *controller.Client) (*Console, error) {
	rl, err := readline.New(promptNormal)
	if err != nil {
		return nil, err
	}

	bf, bfc, err := client.Bifrost(ctx)
	if err != nil {
		return nil, err
	}

	return &Console{
		client:  client,
		bf:      bf,
		bclient: bfc,
		tok:     msgproto.NewTokeniser(),
		rl:      rl,
	}, nil
}

// Close cleans up a Console after it's done.
func (c *Console) Close() error {
	return c.rl.Close()
}

// Run spins up the Console goroutines, and waits for them to terminate.
// It returns any errors returned while closing the Console.
func (c *Console) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	var err error

	// There is seemingly no easy way of making the transmission loop close gracefully;
	// we consequently don't add it to the wait group.
	wg.Add(2)
	go func() {
		c.bf.Run(ctx)
		wg.Done()
	}()
	go func() {
		c.runTx(ctx)
		// See above
	}()
	go func() {
		c.runRx()
		err = c.Close()
		wg.Done()
	}()

	wg.Wait()
	return err
}

// runRx runs the Console's message receiver loop.
// This prints messages to stdout.
func (c *Console) runRx() {
	// We don't have to check c.bclient.Done here:
	// client always drops both Rx and Done when shutting down.
	for m := range c.bclient.Rx {
		mbytes, err := m.Pack()
		if err != nil {
			c.outputError(err)
			continue
		}

		if err := c.outputMessage(mbytes); err != nil {
			c.outputError(err)
		}
	}
}

// runTx runs the Console's message transmitter loop.
// This reads from stdin.
func (c *Console) runTx(ctx context.Context) {
	c.txrun = true
	for c.txrun {
		line, terr := c.rl.Readline()

		if terr != nil {
			c.outputError(terr)
			return
		}

		needMore := c.handleRawLine(ctx, lineToTerminatedBytes(line))
		if needMore {
			c.rl.SetPrompt(promptContinue)
		} else {
			c.rl.SetPrompt(promptNormal)
		}
	}
}

// lineToTerminatedBytes turns a line string, less a newline, to a byte array with a newline.
func lineToTerminatedBytes(line string) []byte {
	var sbuf bytes.Buffer
	sbuf.WriteString(line)
	sbuf.WriteRune('\n')
	return sbuf.Bytes()
}

func (c *Console) handleRawLine(ctx context.Context, bytes []byte) bool {
	pos := 0
	nbytes := len(bytes)
	for pos < nbytes {
		nread, lineok, line := c.tok.TokeniseBytes(bytes[pos:])
		if !lineok {
			return true
		}

		pos += nread

		clientok, err := c.handleLine(ctx, line)
		// TODO(@MattWindsor91): handle txrun better?
		c.txrun = c.txrun && clientok

		if err != nil {
			c.outputError(err)
		}
	}

	return false
}

// handleLine interprets a line (word array) as a console command.
// The line should have been tokenised using Bifrost tokenisation rules.
// If the line is a special command (starts with /), it is handled accordingly.
// Otherwise, it is considered a tagless Bifrost message.
//
// Returns whether the upstream client is still taking messages, and any errors
// arising from processing the line.
func (c *Console) handleLine(ctx context.Context, line []string) (bool, error) {
	if 0 == len(line) {
		return true, nil
	}

	if scword, issc := parseSpecialCommand(line[0]); issc {
		return c.handleSpecialCommand(ctx, scword, line[1:])
	}

	return c.handleBifrostLine(ctx, line)
}

// handleBifrostLine interprets a line (word array) as a tagless Bifrost
// message.
// The line should have been tokenised using Bifrost tokenisation rules.
//
// Returns whether the upstream client is still taking messages, and any errors
// arising from processing the line.
func (c *Console) handleBifrostLine(ctx context.Context, line []string) (bool, error) {
	tag, err := msgproto.NewTag()
	if err != nil {
		return true, err
	}

	tline := make([]string, len(line)+1)
	tline[0] = tag
	copy(tline[1:], line)
	return c.txLine(ctx, tline)
}

func (c *Console) txLine(ctx context.Context, line []string) (bool, error) {
	msg, merr := msgproto.LineToMessage(line)
	if merr != nil {
		return true, merr
	}

	return c.bclient.Send(ctx, *msg), nil
}

// handleSpecialCommand handles special command word scword with arguments args.
// It returns a Boolean reporting whether the client is still taking messages,
// and any errors that occur during processing.
func (c *Console) handleSpecialCommand(ctx context.Context, scword string, args []string) (bool, error) {
	switch scword {
	case "quit":
		return false, c.handleQuit(ctx, args)
	case "tag":
		// Send message with specific tag
		return c.txLine(ctx, args)
	default:
		return true, fmt.Errorf("unknown sc")
	}
}

// handleQuit handles a quit message.
func (c *Console) handleQuit(ctx context.Context, args []string) error {
	if 0 != len(args) {
		return fmt.Errorf("bad arity")
	}

	c.txrun = false
	return c.client.Shutdown(ctx)
}

// parseSpecialCommand tries to interpret word as a special command.
// If word is a special command, it returns the word less the special-command prefix, and true.
// Else, it returns an undefined string, and false.
func parseSpecialCommand(word string) (string, bool) {
	if strings.HasPrefix(word, "/") {
		return word[1:], true
	}
	return "", false
}

// outputMessage outputs a packed message to stdout.
func (c *Console) outputMessage(mbytes []byte) error {
	buf := bytes.NewBufferString(prefixMessage)
	if _, err := buf.WriteRune(' '); err != nil {
		return err
	}
	// mbytes will include the newline.
	if _, err := buf.Write(mbytes); err != nil {
		return err
	}
	_, err := buf.WriteTo(c.rl.Stdout())
	return err
}

// outputError prints an error e to stderr.
func (c *Console) outputError(e error) {
	if _, err := fmt.Fprintln(c.rl.Stderr(), prefixError, e.Error()); err != nil {
		fmt.Println("error when writing to stderr (!):", err.Error())
	}
}
