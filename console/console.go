// Package console is a simple console for inputting Bifrost commands.
package console

import (
	"bytes"
	"fmt"

	"github.com/UniversityRadioYork/baps3d/bifrost"
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
	requestTx  chan<- bifrost.Message
	responseRx <-chan bifrost.Message
	tok        *bifrost.Tokeniser
	rl         *readline.Instance
}

// New creates a new Console.
// This can fail if the underlying console library fails.
func New(requestTx chan<- bifrost.Message, responseRx <-chan bifrost.Message) (*Console, error) {
	rl, err := readline.New(promptNormal)
	if err != nil {
		return nil, err
	}

	return &Console{
		requestTx:  requestTx,
		responseRx: responseRx,
		tok:        bifrost.NewTokeniser(),
		rl:         rl,
	}, nil
}

// Close cleans up a Console after it's done.
func (c *Console) Close() {
	c.rl.Close()
}

// RunRx runs the Console's message receiver loop.
// This prints messages to stdout.
func (c *Console) RunRx() {
	for m := range c.responseRx {
		mbytes, err := m.Pack()
		if err != nil {
			c.outputError(err)
			return
		}

		c.outputMessage(mbytes)
	}
}

// RunTx runs the Console's message transmitter loop.
// This reads from stdin.
func (c *Console) RunTx() {
	for {
		string, terr := c.rl.Readline()

		if terr != nil {
			c.outputError(terr)
			return
		}

		// Readline doesn't give us the newline
		var sbuf bytes.Buffer
		sbuf.WriteString(string)
		sbuf.WriteRune('\n')

		needMore := c.tokenise(sbuf.Bytes())
		if needMore {
			c.rl.SetPrompt(promptContinue)
		} else {
			c.rl.SetPrompt(promptNormal)
		}
	}
}

func (c *Console) tokenise(bytes []byte) bool {
	pos := 0
	nbytes := len(bytes)
	for pos < nbytes {
		nread, lineok, line := c.tok.TokeniseBytes(bytes[pos:])
		if !lineok {
			return true
		}

		pos += nread
		c.txMessage(line)
	}

	return false
}

func (c *Console) txMessage(line []string) {
	msg, merr := bifrost.LineToMessage(line)
	if merr != nil {
		c.outputError(merr)
		return
	}

	c.requestTx <- *msg
}

// outputMessage outputs a packed message to stdout.
func (c *Console) outputMessage(mbytes []byte) {
	var err error
	buf := bytes.NewBufferString(prefixMessage)
	if _, err = buf.WriteRune(' '); err != nil {
		c.outputError(err)
		return
	}
	// mbytes will include the newline.
	if _, err = buf.Write(mbytes); err != nil {
		c.outputError(err)
		return
	}
	if _, err = buf.WriteTo(c.rl.Stdout()); err != nil {
		c.outputError(err)
		return
	}
}

// outputError prints an error e to stderr.
func (c *Console) outputError(e error) {
	fmt.Fprintln(c.rl.Stderr(), prefixError, e.Error())
}
