package msgproto

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

const (
	// Standard Bifrost message word constants.

	// - Requests

	// - Responses

	// RsAck denotes a message with the 'ACK' response.
	RsAck string = "ACK"
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

// AddArg adds the given arguments to a Message in-place.
// The given Message-pointer is returned, to allow for chaining.
func (m *Message) AddArgs(args ...string) *Message {
	m.args = append(m.args, args...)
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
func (m *Message) Pack() ([]byte, error) {
	buf, err := m.packToBuffer()
	if err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

func (m *Message) packToBuffer() (*bytes.Buffer, error) {
	output := bytes.NewBufferString(m.tag + " " + m.word)

	for _, a := range m.args {
		a = m.escapeArgIfNeeded(a)

		if _, err := output.WriteString(" " + a); err != nil {
			return output, err
		}
	}
	_, err := output.WriteRune('\n')
	return output, err
}

func (m *Message) escapeArgIfNeeded(a string) string {
	for _, c := range a {
		if c < unicode.MaxASCII && (unicode.IsSpace(c) || strings.ContainsRune(`'"\`, c)) {
			return escapeArgument(a)
		}
	}
	return a
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
func (m *Message) Arg(index int) (string, error) {
	if index < 0 {
		return "", fmt.Errorf("got negative index %d", index)
	}

	if len(m.args) <= index {
		return "", fmt.Errorf("wanted argument %d, only %d arguments", index, len(m.args))
	}

	return m.args[index], nil
}

// String returns a string representation of a Message.
// This isn't necessarily the wire representation: use Pack instead.
func (m *Message) String() string {
	buf, err := m.packToBuffer()
	if err != nil {
		return "(error)"
	}
	return buf.String()
}

// LineToMessage constructs a Message struct from a line of word-strings.
func LineToMessage(line []string) (*Message, error) {
	if len(line) < 2 {
		return nil, fmt.Errorf("insufficient words")
	}

	msg := NewMessage(line[0], line[1]).AddArgs(line[2:]...)
	return msg, nil
}
