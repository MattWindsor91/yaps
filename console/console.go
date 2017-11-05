// Package console is a simple console for inputting Bifrost commands.
package console

import (
	"fmt"
	"os"

	"github.com/UniversityRadioYork/baps3d/bifrost"
)

type Console struct {
	requestTx  chan<- bifrost.Message
	responseRx <-chan bifrost.Message
	in         *bifrost.ReaderTokeniser
}

func New(requestTx chan<- bifrost.Message, responseRx <-chan bifrost.Message) *Console {
	return &Console{
		requestTx:  requestTx,
		responseRx: responseRx,
		in:         bifrost.NewReaderTokeniser(os.Stdin),
	}
}

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

func (c *Console) RunTx() {
	for {
		line, terr := c.in.ReadLine()
		if terr != nil {
			fmt.Println("-> got error:", terr)
			return
		}

		msg, merr := bifrost.LineToMessage(line)
		if merr != nil {
			fmt.Println("-> invalid message:", merr)
			continue
		}

		c.requestTx <- *msg
	}
}
