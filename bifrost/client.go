package bifrost

// File bifrost/client.go describes clients that communicate at the level of Bifrost messages.

// Client is a struct containing channels used to talk to a Bifrost endpoint.
type Client struct {
	// ReqTx is the channel for transmitting requests.
	ReqTx chan<- Message

	// ResRx is the channel for receiving responses.
	ResRx <-chan Message

	// Done is a channel that is closed when the endpoint has shut down.
	Done <-chan struct{}
}

// Send tries to send a request on a BifrostClient.
// It returns false if the BifrostClient's upstream has shut down.
//
// Send is just sugar over a Select between ReqTx and Done, and it is
// ok to do this manually using the channels themselves.
func (c *Client) Send(r Message) bool {
	select {
	case <-c.Done:
		return false
	case c.ReqTx <- r:
	}
	return true
}

// Endpoint contains the opposite end of a Client's channels.
type Endpoint struct {
	// ReqRx is the channel for receiving requests.
	ReqRx <-chan Message

	// ResTx is the channel for sending responses.
	ResTx chan<- Message

	// Done is a channel to be closed when the endpoint wants to shut down.
	Done chan<- struct{}
}

// Close closes all of c's transmission channels.
func (c *Endpoint) Close() {
	close(c.ResTx);
	close(c.Done);
}

// NewClient creates a pair of Bifrost client channel sets.
func NewClient() (*Client, *Endpoint) {
	res := make(chan Message)
	req := make(chan Message)
	done := make(chan struct{})

	client := Client{
		ResRx: res,
		ReqTx: req,
		Done:  done,
	}

	coclient := Endpoint{
		ResTx: res,
		ReqRx: req,
		Done:  done,
	}

	return &client, &coclient
}
