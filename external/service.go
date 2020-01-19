package external

import (
	"errors"
	"github.com/UniversityRadioYork/baps3d/bifrost"
	"net"
)

// Service is a Controllable that delegates requests and responses to a Bifrost service.
type Service struct {
	// role stores the last known role of the client.
	role string

	// io represents the connection to the external service.
	io bifrost.IoClient
}

func (c Service) ParseBifrostRequest(word string, args []string) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (c Service) EmitBifrostResponse(tag string, resp interface{}, out chan<- bifrost.Message) error {
	return errors.New("not implemented")
}

// NewService connects to a Bifrost server at address, and, if successful, constructs a new ExternalService over it.
func NewService(address string) (c *Service, err error) {
	var conn net.Conn
	if conn, err = net.Dial("tcp", address); err != nil {
		return nil, err
	}

	bcl, bep := bifrost.NewEndpointPair()

	var role string
	if role, err = handshake(bep); err != nil {
		return nil, err
	}

	c = &Service{role: role, io: bifrost.IoClient{Bifrost:bcl, Conn: conn}}
	return c, nil
}

func handshake(endpoint *bifrost.Endpoint) (role string, err error) {
	return "", errors.New("not implemented")
}
