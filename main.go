package main

import "github.com/UniversityRadioYork/baps3d/list"

func spinUpList() (*list.Controller, *list.Client) {
	lst := list.New()
	return list.NewController(lst)
}

func main() {
	lc, cli := spinUpList()
	go lc.Run()

	close(cli.Tx)
	for {
		_, ok := <-cli.Rx
		// TODO: do something with this
		if !ok {
			break
		}
	}
}
