package bifrost

// File bifrost/client.go describes clients that communicate at the level of Bifrost messages.

// Note: we use the Client and Endpoint structs in both sides of a client/server communication,
// hence why their channels are called Tx and Rx and not something more indicative (eg 'RequestTx' or 'ResponseRx').

// Client is a struct containing channels used to talk to a Bifrost endpoint.
type Client struct {
	// Tx is the channel for transmitting messages to the endpoint.
	Tx chan<- Message

	// Rx is the channel for receiving messages from the endpoint.
	Rx <-chan Message

	// Done is a channel that is closed when the endpoint has shut down.
	Done <-chan struct{}
}

// Send tries to send a request on a BifrostClient.
// It returns false if the BifrostClient's upstream has shut down.
//
// Send is just sugar over a Select between Tx and Done, and it is
// ok to do this manually using the channels themselves.
func (c *Client) Send(r Message) bool {
	select {
	case <-c.Done:
		return false
	case c.Tx <- r:
	}
	return true
}

// Endpoint contains the opposite end of a Client's channels.
type Endpoint struct {
	// Rx is the channel for receiving messages intended for the endpoint.
	Rx <-chan Message

	// Tx is the channel for transmitting messages from the endpoint.
	Tx chan<- Message

	// Done is a channel to be closed when the endpoint wants to shut down.
	Done chan<- struct{}
}

// Close closes all of c's transmission channels.
func (c *Endpoint) Close() {
	close(c.Tx);
	close(c.Done);
}

// NewClient creates a pair of Bifrost client channel sets.
func NewClient() (*Client, *Endpoint) {
	res := make(chan Message)
	req := make(chan Message)
	done := make(chan struct{})

	client := Client{
		Rx:   res,
		Tx:   req,
		Done: done,
	}

	endpoint := Endpoint{
		Tx:   res,
		Rx:   req,
		Done: done,
	}

	return &client, &endpoint
}
