package message

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

const (
	// Standard Bifrost message word constants.

	// - Requests

	// RqRead denotes a 'read' request message.
	RqRead string = "read"

	// RqWrite denotes a 'write' request message.
	RqWrite string = "write"

	// RqDelete denotes a 'delete' request message.
	RqDelete string = "delete"

	// - Responses

	// RsRes denotes a message with the 'RES' response.
	RsRes string = "RES"

	// RsUpdate denotes a message with the 'UPDATE' response.
	RsUpdate string = "UPDATE"

	// RsAck denotes a message with the 'ACK' response.
	RsAck string = "ACK"

	// RsOhai denotes a message with the 'OHAI' response.
	RsOhai string = "OHAI"
)

// Message is a structure representing a full BAPS3 message.
// It is comprised of a word, which is stored as a string, and zero or
// more string arguments.
type Message struct {
	word string
	args []string
}

// New creates and returns a new Message with the given message word.
// The message will initially have no arguments; use AddArg to add arguments.
func New(word string) *Message {
	return &Message{
		word: word,
	}
}

// AddArg adds the given argument to a Message in-place.
// The given Message-pointer is returned, to allow for chaining.
func (m *Message) AddArg(arg string) *Message {
	m.args = append(m.args, arg)
	return m
}

func escapeArgument(input string) string {
	return "'" + strings.Replace(input, "'", `'\''`, -1) + "'"
}

// Pack outputs the given Message as raw bytes representing a BAPS3 message.
// These bytes can be sent down a TCP connection to a BAPS3 server, providing
// they are terminated using a line-feed character.
func (m *Message) Pack() (packed []byte, err error) {
	output := new(bytes.Buffer)

	_, err = output.WriteString(m.word)
	if err != nil {
		return
	}

	for _, a := range m.args {
		// Escape arg if needed
		for _, c := range a {
			if c < unicode.MaxASCII && (unicode.IsSpace(c) || strings.ContainsRune(`'"\`, c)) {
				a = escapeArgument(a)
				break
			}
		}

		_, err = output.WriteString(" " + a)
		if err != nil {
			return
		}
	}
	output.WriteString("\n")

	packed = output.Bytes()
	return
}

// Word returns the message word of the given Message.
func (m *Message) Word() string {
	return m.word
}

// Args returns the slice of Arguments.
func (m *Message) Args() []string {
	return m.args
}

// Arg returns the index-th argument of the given Message.
// The first argument is argument 0.
// If the argument does not exist, an error is returned via err.
func (m *Message) Arg(index int) (arg string, err error) {
	if index < 0 {
		err = fmt.Errorf("got negative index %d", index)
	} else if len(m.args) <= index {
		err = fmt.Errorf("wanted argument %d, only %d arguments", index, len(m.args))
	} else {
		arg = m.args[index]
	}
	return
}

// String returns a string representation of a Message.
// This is not the wire representation: use Pack instead.
func (m *Message) String() (outstr string) {
	outstr = m.word
	for _, s := range m.args {
		outstr += " " + s
	}
	return
}

// lineToMessage constructs a Message struct from a line of word-strings.
func LineToMessage(line []string) (msg *Message, err error) {
	if len(line) == 0 {
		err = fmt.Errorf("cannot construct message from zero words")
	} else {
		msg = New(line[0])
		for _, arg := range line[1:] {
			msg.AddArg(arg)
		}
	}

	return
}
