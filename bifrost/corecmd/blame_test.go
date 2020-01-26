package corecmd

import "fmt"

// ExampleBlame_String gives examples for the use of String on the Blame type.
func ExampleBlame_String() {
	fmt.Println(BlameClient.String())
	fmt.Println(BlameServer.String())
	fmt.Println(BlameUnknown.String())

	// The zero value for Blame is also unknown:
	var u Blame
	fmt.Println(u.String())

	// Output:
	// client
	// server
	// unknown
	// unknown
}
