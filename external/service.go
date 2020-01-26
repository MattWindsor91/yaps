package external

import (
	"context"
	"errors"
	"net"

	"github.com/UniversityRadioYork/baps3d/bifrost"
	"github.com/UniversityRadioYork/baps3d/bifrost/corecmd"
	"github.com/UniversityRadioYork/baps3d/bifrost/msgproto"
	"github.com/UniversityRadioYork/baps3d/controller"
)

// Service is a Controllable that delegates requests and responses to a Bifrost service.
type Service struct {
	// role stores the last known role of the client.
	role string

	// io represents the connection to the external service.
	io bifrost.IoClient
}

func (s *Service) RoleName() string {
	return s.role
}

func (s *Service) Dump(ctx context.Context, dumpCb controller.ResponseCb) {
	panic("implement me")
}

func (s *Service) HandleRequest(replyCb controller.ResponseCb, bcastCb controller.ResponseCb, rbody interface{}) error {
	panic("implement me")
}

func (c *Service) ParseBifrostRequest(word string, args []string) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (c *Service) EmitBifrostResponse(tag string, resp interface{}, out chan<- msgproto.Message) error {
	return errors.New("not implemented")
}

// NewService connects to a Bifrost server at address, and, if successful, constructs a new ExternalService over it.
func NewService(address string) (c *Service, err error) {
	var conn net.Conn
	if conn, err = net.Dial("tcp", address); err != nil {
		return nil, err
	}

	srvEnd, cliEnd := bifrost.NewEndpointPair()

	var role string
	if role, err = handshake(cliEnd); err != nil {
		return nil, err
	}

	c = &Service{role: role, io: bifrost.IoClient{Bifrost: srvEnd, Conn: conn}}
	return c, nil
}

// handshake performs the Bifrost handshake with whichever Bifrost service is on the other end of cliEnd.
func handshake(cliEnd *bifrost.Endpoint) (role string, err error) {
	// TODO(@MattWindsor91): make this more symmetric with the way it's done on the client side
	ohaiMsg := <-cliEnd.Rx
	if _, err := corecmd.ParseOhaiResponse(&ohaiMsg); err != nil {
		return "", err
	}

	return "", errors.New("not implemented")
}
