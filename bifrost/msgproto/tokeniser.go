package msgproto

import (
	"io"
	"unicode"
)

// quoteType represents one of the types of quoting used in the Bifrost protocol.
type quoteType int

const (
	// none represents the state between quoted parts of a Bifrost message.
	none quoteType = iota

	// single represents 'single quoted' parts of a Bifrost message.
	single

	// double represents "double quoted" parts of a Bifrost message.
	double
)

// ReaderTokeniser adapts a Tokeniser to deal with a Reader.
type ReaderTokeniser struct {
	tok    *Tokeniser
	reader io.Reader
	buf    [4096]byte
	pos    int
	max    int
}

// tokeniseUntilLine drains t's internal buffer into its tokeniser until it runs out or produces a line.
func (t *ReaderTokeniser) tokeniseUntilLine() (line []string, lineok bool) {
	var nread int
	for t.pos < t.max && !lineok {
		nread, lineok, line = t.tok.TokeniseBytes(t.buf[t.pos:t.max])
		t.pos += nread
	}
	return
}

// fillFromReader fills t's internal buffer using its reader.
// It can fail with errors from the reader.
func (t *ReaderTokeniser) fillFromReader() (err error) {
	t.pos = 0
	t.max, err = t.reader.Read(t.buf[:])
	return
}

// ReadLine reads a tokenised line from the Reader.
// ReadLine may return an error if the Reader chokes.
func (t *ReaderTokeniser) ReadLine() ([]string, error) {
	for {
		if line, lineok := t.tokeniseUntilLine(); lineok {
			return line, nil
		}
		if err := t.fillFromReader(); err != nil {
			return []string{}, err
		}
	}
}

// NewReaderTokeniser creates and returns a new, empty ReaderTokeniser.
// The ReaderTokeniser will read from the given Reader when Tokenise is called.
func NewReaderTokeniser(reader io.Reader) *ReaderTokeniser {
	return &ReaderTokeniser{
		tok:    NewTokeniser(),
		reader: reader,
		pos:    0,
		max:    0,
	}
}

// Tokeniser holds the state of a Bifrost protocol tokeniser.
type Tokeniser struct {
	inWord           bool
	escapeNextChar   bool
	currentQuoteType quoteType
	word             []byte
	words            []string
}

// NewTokeniser creates and returns a new, empty Tokeniser.
func NewTokeniser() *Tokeniser {
	return &Tokeniser{
		escapeNextChar:   false,
		currentQuoteType: none,
		word:             []byte{},
		inWord:           false,
		words:            []string{},
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

// TokeniseBytes tokenises an array of bytes.
// It returns the number of bytes read, whether or not it read a line, and the line contents if true.
func (t *Tokeniser) TokeniseBytes(bs []byte) (nread int, lineok bool, line []string) {
	nread = 0
	lineok = false

	if len(bs) == 0 {
		return
	}

	for i := 0; i < len(bs); i++ {
		if t.tokeniseByte(bs[i]) {
			nread = i + 1
			lineok = true
			line = t.words
			t.words = []string{}
			return
		}
	}

	return
}

// tokeniseByte tokenises a single byte b.
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
