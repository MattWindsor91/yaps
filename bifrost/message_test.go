package bifrost

import "testing"
import "reflect"

func TestMessage(t *testing.T) {
	cases := []struct {
		words []string
		msg   *Message
	}{
		// Empty request
		{
			[]string{"write"},
			NewMessage("write"),
		},
		// Request with one argument
		{
			[]string{"read", "/control/state"},
			NewMessage("read").AddArg("/control/state"),
		},
		// Request with multiple argument
		{
			[]string{"write", "/player/time", "0"},
			NewMessage("write").AddArg("/player/time").AddArg("0"),
		},
		// Empty response
		{
			[]string{"RES"},
			NewMessage("RES"),
		},
		// Response with one argument
		{
			[]string{"OHAI", "playd 1.0.0"},
			NewMessage(RsOhai).AddArg("playd 1.0.0"),
		},
		// Response with multiple argument
		{
			[]string{"ACK", "int", "OK", "1337"},
			NewMessage(RsAck).AddArg("int").AddArg("OK").AddArg("1337"),
		},
	}

	for _, c := range cases {
		if c.words[0] != c.msg.Word() {
			t.Errorf("Word() == %q, expected %q", c.msg.Word(), c.words[0])
		}
	}

	// And now, test args.
	// TODO(CaptainHayashi): refactor the above to integrate this test
	args := []string{"bibbity", "bobbity", "boo"}
	msg := NewMessage("flax")
	for _, arg := range args {
		msg.AddArg(arg)
	}

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

	for _, c := range cases {
		got, err := LineToMessage(c.words)
		if err != nil {
			t.Errorf("unexpected error with: %q", got)
		} else if !reflect.DeepEqual(got, c.msg) {
			t.Errorf("Got %q, wanted %q", got, c.msg)
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
			&Message{"write", []string{"uuid", "/player/file", "/home/donald/wjaz.mp3"}},
			[]byte("write uuid /player/file /home/donald/wjaz.mp3\n"),
		},
		// Backslashes
		{
			&Message{"write", []string{"uuid", "/player/file", `C:\silly\windows\is\silly`}},
			[]byte(`write uuid /player/file 'C:\silly\windows\is\silly'` + "\n"),
		},
		// No args TODO: Can't happen any more?
		{
			&Message{"read", []string{}},
			[]byte("read\n"),
		},
		// Spaces
		{
			&Message{"write", []string{"uuid", "/player/file", "/home/donald/01 The Nightfly.mp3"}},
			[]byte("write uuid /player/file '/home/donald/01 The Nightfly.mp3'\n"),
		},
		// Single quotes
		{
			&Message{RsOhai, []string{"a'bar'b"}},
			[]byte(`OHAI 'a'\''bar'\''b'` + "\n"),
		},
		// Double quotes
		{
			&Message{RsOhai, []string{`a"bar"b`}},
			[]byte(`OHAI 'a"bar"b'` + "\n"),
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
