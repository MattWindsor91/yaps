package tokeniser

import (
	"io"
	"unicode"
)

// quoteType represents one of the types of quoting used in the BAPS3 protocol.
type quoteType int

const (
	// none represents the state between quoted parts of a BAPS3 message.
	none quoteType = iota

	// single represents 'single quoted' parts of a BAPS3 message.
	single

	// double represents "double quoted" parts of a BAPS3 message.
	double
)

// Tokeniser holds the state of a Bifrost protocol tokeniser.
type Tokeniser struct {
	inWord           bool
	escapeNextChar   bool
	currentQuoteType quoteType
	word             []byte
	words            []string
	reader           io.Reader
}

// NewTokeniser creates and returns a new, empty Tokeniser.
// The Tokeniser will read from the given Reader when Tokenise is called.
func New(reader io.Reader) *Tokeniser {
	return &Tokeniser{
		escapeNextChar:   false,
		currentQuoteType: none,
		word:             []byte{},
		inWord:           false,
		words:            []string{},
		reader:           reader,
	}
}

func (t *Tokeniser) endWord() {
	if !t.inWord {
		// Don't add an empty word.
		return
	}

	t.words = append(t.words, string(t.word))
	t.word = []byte{}
	t.inWord = false
}

// Tokenise reads a tokenised line from the Reader.
//
// Tokenise may return an error if the Reader chokes.
func (t *Tokeniser) Tokenise() ([]string, error) {
	for {
		abyte, err := t.readByte()
		if err != nil {
			return []string{}, err
		}

		lineDone := t.tokeniseByte(abyte)
		// Have we finished a line?
		// If so, clean up for another tokenising, and return it.
		if lineDone {
			line := t.words
			t.words = []string{}
			return line, nil
		}
	}
}

// readByte pulls a single byte out of the Reader.
// It spins until a successful write or error has been received.
// It then the byte read and nil, or undefined and an error, respectively.
func (t *Tokeniser) readByte() (b byte, err error) {
	// As per http://grokbase.com/t/gg/golang-nuts/139fgmycba
	var bs [1]byte

	// Technically inefficient, but this will be done on network
	// connections mainly anyway, so this shouldn't be the
	// bottleneck.
	for n := 0; n == 0 && err == nil; {
		n, err = t.reader.Read(bs[:])
	}

	b = bs[0]
	return
}

// tokeniseByte tokenises a single byte.
// It returns true if we've finished a line, which can only occur outside of
// quotes
func (t *Tokeniser) tokeniseByte(b byte) bool {
	if t.escapeNextChar {
		t.put(b)
		t.escapeNextChar = false
		return false
	}

	funcs := map[quoteType]func(b byte) bool{
		none:   t.tokeniseNoQuotes,
		single: t.tokeniseSingleQuotes,
		double: t.tokeniseDoubleQuotes,
	}

	return funcs[t.currentQuoteType](b)
}

// tokeniseNoQuotes tokenises a single byte outside quote characters.
// It returns true if we've finished a line, and any error that occurred while
// tokenising.
func (t *Tokeniser) tokeniseNoQuotes(b byte) bool {
	switch b {
	case '\'':
		// Switching into single quotes mode starts a word.
		// This is to allow '' to represent the empty string.
		t.inWord = true
		t.currentQuoteType = single
	case '"':
		// Switching into double quotes mode starts a word.
		// This is to allow "" to represent the empty string.
		t.inWord = true
		t.currentQuoteType = double
	case '\\':
		t.escapeNextChar = true
	case '\n':
		// We're ending the current word as well as a line.
		t.endWord()
		return true
	default:
		// Note that this will only check for ASCII
		// whitespace, because we only pass it one byte
		// and non-ASCII whitespace is >1 UTF-8 byte.
		if unicode.IsSpace(rune(b)) {
			t.endWord()
		} else {
			t.put(b)
		}
	}

	return false
}

// tokeniseSingleQuotes tokenises a single byte within single quotes.
// We can't finish a line in quotes, so it always returns false.
func (t *Tokeniser) tokeniseSingleQuotes(b byte) bool {
	switch b {
	case '\'':
		t.currentQuoteType = none
	default:
		t.put(b)
	}

	return false
}

// tokeniseDoubleQuotes tokenises a single byte within double quotes.
// We can't finish a line in quotes, so it always returns false.
func (t *Tokeniser) tokeniseDoubleQuotes(b byte) bool {
	switch b {
	case '"':
		t.currentQuoteType = none
	case '\\':
		t.escapeNextChar = true
	default:
		t.put(b)
	}

	return false
}

// put adds a byte to the Tokeniser's word.
func (t *Tokeniser) put(b byte) {
	t.inWord = true
	t.word = append(t.word, b)
}
