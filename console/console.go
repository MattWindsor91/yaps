// Package console is a simple console for inputting Bifrost commands.
package console

import (
	"fmt"
	"os"
		
	"github.com/UniversityRadioYork/baps3d/bifrost"
)

type Console struct {
	msg chan bifrost.Message
	in *bifrost.Tokeniser
}

func New(msg chan bifrost.Message) *Console {
	return &Console{ msg: msg, in: bifrost.NewTokeniser(os.Stdin), }
}

func (c *Console) RunRx() {
}

func (c *Console) RunTx() {
	for {
		line, terr := c.in.Tokenise()
		if terr != nil {
			fmt.Println("-> got error:", terr)
			return
		}

		_, merr := bifrost.LineToMessage(line)
		if merr != nil {
			fmt.Println("-> invalid message:", merr)
			continue
		}

		// TODO: do something with msg
	}
}
