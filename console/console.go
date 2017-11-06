// Package console is a simple console for inputting Bifrost commands.
package console

import (
	"bytes"
	"fmt"
	"os"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/chzyer/readline"
)

const (
	// promptNormal is the normal prompt used by the console.
	promptNormal = "$ "
	// promptContinue is the prompt used when in the middle of a quoted Bifrost message word.
	promptContinue = "> "
)

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
			fmt.Println("-> rx error:", err)
			continue
		}
		os.Stdout.Write(mbytes)
	}
}

// RunTx runs the Console's message transmitter loop.
// This reads from stdin.
func (c *Console) RunTx() {
	for {
		string, terr := c.rl.Readline()
		if terr != nil {
			// TODO(@MattWindsor91): send to rx
			fmt.Println("-> got error:", terr)
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
		// TODO(@MattWindsor91): send to rx
		fmt.Println("-> invalid message:", merr)
		return
	}

	c.requestTx <- *msg
}
