package bifrost

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

const (
	// Standard Bifrost message word constants.

	// TagBcast is the tag used for broadcasts.
	TagBcast string = "!"

	// TagUnknown is the tag used for when we don't know the right tag to use.
	TagUnknown string = "?"

	// - Requests

	// - Responses

	// RsAck denotes a message with the 'ACK' response.
	RsAck string = "ACK"

	// RsOhai denotes a message with the 'OHAI' response.
	RsOhai string = "OHAI"
)

// Message is a structure representing a full BAPS3 message.
// It is comprised of a string tag, a string word, and zero or
// more string arguments.
type Message struct {
	tag  string
	word string
	args []string
}

// NewMessage creates and returns a new Message with the given tag and message word.
// The message will initially have no arguments; use AddArg to add arguments.
func NewMessage(tag, word string) *Message {
	return &Message{
		tag:  tag,
		word: word,
	}
}

// AddArg adds the given argument to a Message in-place.
// The given Message-pointer is returned, to allow for chaining.
func (m *Message) AddArg(arg string) *Message {
	m.args = append(m.args, arg)
	return m
}

// escapeArgument escapes a message argument.
// It does so using Bifrost's single-quoting, which is easy to encode but bad for human readability.
func escapeArgument(input string) string {
	return "'" + strings.Replace(input, "'", `'\''`, -1) + "'"
}

// Pack outputs the given Message as raw bytes representing a Bifrost message.
// These bytes can be sent down a TCP connection to a Bifrost server, providing
// they are terminated using a line-feed character.
func (m *Message) Pack() (packed []byte, err error) {
	output := new(bytes.Buffer)

	if _, err = output.WriteString(m.tag + " " + m.word); err != nil {
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

		if _, err = output.WriteString(" " + a); err != nil {
			return
		}
	}
	if _, err = output.WriteString("\n"); err != nil {
		return
	}

	packed = output.Bytes()
	return
}

// Tag returns this Message's tag.
func (m *Message) Tag() string {
	return m.tag
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

// LineToMessage constructs a Message struct from a line of word-strings.
func LineToMessage(line []string) (msg *Message, err error) {
	if len(line) < 2 {
		err = fmt.Errorf("insufficient words")
	} else {
		msg = NewMessage(line[0], line[1])
		for _, arg := range line[2:] {
			msg.AddArg(arg)
		}
	}

	return
}
