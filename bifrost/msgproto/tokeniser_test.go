package msgproto

import (
	"bytes"
	"io"
	"testing"
)

func cmpLines(a [][]string, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, aline := range a {
		if !cmpWords(aline, b[i]) {
			return false
		}
	}

	return true
}

func cmpWords(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, aword := range a {
		if aword != b[i] {
			return false
		}
	}

	return true
}

func TestReaderTokeniser(t *testing.T) {
	// For now, only test one complete line at a time.
	// TODO(CaptainHayashi): add partial-line tests.

	// Tests adapted from (and labelled with respect to):
	// http://universityradioyork.github.io/baps3-spec/comms/internal/protocol.html#examples

	cases := []struct {
		in   string
		want [][]string
	}{
		// E1 - empty string
		{
			"",
			[][]string{},
		},
		// E2 - empty line
		{
			"\n",
			[][]string{{}},
		},
		// E3 - empty single-quoted string
		{
			"''\n",
			[][]string{{""}},
		},
		// E4 - empty double-quoted string
		{
			"\"\"\n",
			[][]string{{""}},
		},
		// W1 - space-delimited words
		{
			"foo bar baz\n",
			[][]string{{"foo", "bar", "baz"}},
		},
		// W2 - tab-delimited words
		{
			"fizz\tbuzz\tpoo\n",
			[][]string{{"fizz", "buzz", "poo"}},
		},
		// W3 - oddly-delimited words
		{
			"bibbity\rbobbity\rboo\n",
			[][]string{{"bibbity", "bobbity", "boo"}},
		},
		// W4 - CRLF tolerance
		{
			"silly windows\r\n",
			[][]string{{"silly", "windows"}},
		},
		// W5 - leading whitespace
		{
			"     abc def\n",
			[][]string{{"abc", "def"}},
		},
		// W6 - trailing whitespace
		{
			"ghi jkl     \n",
			[][]string{{"ghi", "jkl"}},
		},
		// W7 - surrounding whitespace
		{
			"     mno pqr     \n",
			[][]string{{"mno", "pqr"}},
		},
		// Q1 - backslash escaping
		{
			"abc\\\ndef\n",
			[][]string{{"abc\ndef"}},
		},
		// Q2 - double-quoting
		{
			"\"abc\ndef\"\n",
			[][]string{{"abc\ndef"}},
		},
		// Q3 - double-quoting, backslash-escape
		{
			"\"abc\\\ndef\"\n",
			[][]string{{"abc\ndef"}},
		},
		// Q4 - single-quoting
		{
			"'abc\ndef'\n",
			[][]string{{"abc\ndef"}},
		},
		// Q5 - single-quoting, backslash-'escape'
		{
			"'abc\\\ndef'\n",
			[][]string{{"abc\\\ndef"}},
		},
		// Q6 - backslash-escaped double quote
		{
			"Scare\\\" quotes\\\"\n",
			[][]string{{"Scare\"", "quotes\""}},
		},
		// Q7 - backslash-escaped single quote
		{
			"I\\'m free\n",
			[][]string{{"I'm", "free"}},
		},
		// Q8 - single-quoted single quote
		{
			`'hello, I'\''m an escaped single quote'` + "\n",
			[][]string{{"hello, I'm an escaped single quote"}},
		},
		// Q9 - double-quoted single quote
		{
			`"hello, this is an \" escaped double quote"` + "\n",
			[][]string{{`hello, this is an " escaped double quote`}},
		},
		// M1 - multiple lines
		{
			"first line\nsecond line\n",
			[][]string{
				{"first", "line"},
				{"second", "line"},
			},
		},
		// U1 - UTF-8
		{
			"北野 武\n",
			[][]string{{"北野", "武"}},
		},
		// U2 intentionally left blank.
		// X1 - Sample BAPS3 command, with double-quoted Windows path
		{
			`enqueue file "C:\\Users\\Test\\Artist - Title.mp3" 1` + "\n",
			[][]string{
				{"enqueue", "file", `C:\Users\Test\Artist - Title.mp3`, "1"},
			},
		},
	}

	for _, c := range cases {
		br := bytes.NewReader([]byte(c.in))
		tok := NewReaderTokeniser(br)

		var (
			got  [][]string
			err  error
			line []string
		)

		for {
			line, err = tok.ReadLine()

			if err != nil {
				break
			}

			got = append(got, line)
		}

		if err != io.EOF {
			t.Errorf("ReadLine(%q) gave error %q", c.in, err)
		}
		if !cmpLines(got, c.want) {
			t.Errorf("ReadLine(%q) == %q, want %q", c.in, got, c.want)
		}
	}
}
