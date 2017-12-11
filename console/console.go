// Package console is a simple console for inputting Bifrost commands.
package console

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/chzyer/readline"
	"github.com/satori/go.uuid"
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
	txrun      bool
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
	c.txrun = true
	for c.txrun {
		string, terr := c.rl.Readline()

		if terr != nil {
			c.outputError(terr)
			return
		}

		// Readline doesn't give us the newline
		var sbuf bytes.Buffer
		sbuf.WriteString(string)
		sbuf.WriteRune('\n')

		needMore := c.handleRawLine(sbuf.Bytes())
		if needMore {
			c.rl.SetPrompt(promptContinue)
		} else {
			c.rl.SetPrompt(promptNormal)
		}
	}
}

func (c *Console) handleRawLine(bytes []byte) bool {
	pos := 0
	nbytes := len(bytes)
	for pos < nbytes {
		nread, lineok, line := c.tok.TokeniseBytes(bytes[pos:])
		if !lineok {
			return true
		}

		pos += nread
		c.handleLine(line)
	}

	return false
}

func (c *Console) handleLine(line []string) {
	if 0 == len(line) {
		return
	}

	if !c.handleSpecialCommand(line) {
		// Default behaviour: send as Bifrost message, but with unique tag
		tline := make([]string, len(line)+1)
		tline[0] = gentag()
		copy(tline[1:], line)
		c.txLine(tline)
	}
}

// gentag generates a new, (hopefully) unique tag.
func gentag() string {
	return uuid.NewV1().String()
}

func (c *Console) txLine(line []string) {
	msg, merr := bifrost.LineToMessage(line)
	if merr != nil {
		c.outputError(merr)
		return
	}

	c.requestTx <- *msg
}

// handleSpecialCommand tries to interpret line as a special command.
// If line is a special command, it processes line and returns true.
// If not, it returns false and the line should be processed as a raw message.
// line must be non-empty.
func (c *Console) handleSpecialCommand(line []string) bool {
	if scword, issc := parseSpecialCommand(line[0]); issc {
		var err error = nil

		switch scword {
		case "quit":
			// Quit
			if 1 != len(line) {
				err = fmt.Errorf("bad arity")
				break
			}
			c.txrun = false
		case "tag":
			// Send message with specific tag
			c.txLine(line[1:])
		default:
			err = fmt.Errorf("unknown sc")
		}

		if err != nil {
			c.outputError(err)
		}

		return true
	}
	return false
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
