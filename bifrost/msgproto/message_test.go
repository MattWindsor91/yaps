package msgproto

import (
	"testing"
)
import "reflect"

func TestMessage_WordAndTag(t *testing.T) {
	cases := []struct {
		words []string
		msg   *Message
	}{
		// Empty request
		{
			[]string{"x", "write"},
			NewMessage("x", "write"),
		},
		// Request with one argument
		{
			[]string{"y", "read", "/control/state"},
			NewMessage("y", "read").AddArgs("/control/state"),
		},
		// Request with multiple arguments
		{
			[]string{"z", "write", "/player/time", "0"},
			NewMessage("z", "write").AddArgs("/player/time", "0"),
		},
		// Empty response
		{
			[]string{"!", "RES"},
			NewMessage(TagBcast, "RES"),
		},
		// Response with one argument
		{
			[]string{"!", "OHAI", "playd 1.0.0"},
			NewMessage(TagBcast, "OHAI").AddArgs("playd 1.0.0"),
		},
		// Response with multiple argument
		{
			[]string{"x", "ACK", "int", "OK", "1337"},
			NewMessage("x", RsAck).AddArgs("int", "OK", "1337"),
		},
	}

	for _, c := range cases {
		if c.words[0] != c.msg.Tag() {
			t.Errorf("Tag() == %q, expected %q", c.msg.Tag(), c.words[0])
		}
		if c.words[1] != c.msg.Word() {
			t.Errorf("Word() == %q, expected %q", c.msg.Word(), c.words[1])
		}

		got, err := LineToMessage(c.words)
		if err != nil {
			t.Errorf("unexpected error with: %q", got)
		} else if !reflect.DeepEqual(got, c.msg) {
			t.Errorf("Got %q, wanted %q", got, c.msg)
		}
	}
}

func TestMessage_Args(t *testing.T) {
	args := []string{"bibbity", "bobbity", "boo"}
	msg := NewMessage("spelt", "flax").AddArgs(args...)

	// Bounds checking
	for _, i := range []int{-1, len(args)} {
		if _, err := msg.Arg(i); err == nil {
			t.Errorf("Managed to get %dth arg of a %d-arged Message", i, len(args))
		}
	}

	for i, want := range args {
		got, err := msg.Arg(i)
		if err != nil {
			t.Errorf("unexpected error with Arg(%d)", i)
		} else if got != want {
			t.Errorf("Arg(%d) = %q, want %q", i, got, want)
		}
	}
}

func TestPack(t *testing.T) {
	cases := []struct {
		msg  *Message
		want []byte
	}{
		// Unescaped command
		{
			&Message{"x", "write", []string{"uuid", "/player/file", "/home/donald/wjaz.mp3"}},
			[]byte("x write uuid /player/file /home/donald/wjaz.mp3\n"),
		},
		// Backslashes
		{
			&Message{"y", "write", []string{"uuid", "/player/file", `C:\silly\windows\is\silly`}},
			[]byte(`y write uuid /player/file 'C:\silly\windows\is\silly'` + "\n"),
		},
		// No args TODO: Can't happen any more?
		{
			&Message{"z", "read", []string{}},
			[]byte("z read\n"),
		},
		// Spaces
		{
			&Message{"abc", "write", []string{"uuid", "/player/file", "/home/donald/01 The Nightfly.mp3"}},
			[]byte("abc write uuid /player/file '/home/donald/01 The Nightfly.mp3'\n"),
		},
		// Single quotes
		{
			&Message{TagBcast, "OHAI", []string{"a'bar'b"}},
			[]byte(`! OHAI 'a'\''bar'\''b'` + "\n"),
		},
		// Double quotes
		{
			&Message{TagBcast, "OHAI", []string{`a"bar"b`}},
			[]byte(`! OHAI 'a"bar"b'` + "\n"),
		},
	}

	for _, c := range cases {
		got, err := c.msg.Pack()
		if err != nil {
			t.Errorf("Message.Pack(%q) encountered error %q", c.msg, err)
		}
		if !reflect.DeepEqual(c.want, got) {
			t.Errorf("Message.Pack(%q) == %q, want %q", c.msg, got, c.want)
		}
	}
}

/*
Helper functions for messages
*/
