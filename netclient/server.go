package netclient

import (
	"log"
	"net"
	
	"github.com/UniversityRadioYork/baps3d/comm"
)

// Server holds the internal state of a baps3d TCP server.
type Server struct {
	// l is the Server's logger.
	l *log.Logger

	// host is the Server's host:port string.
	host string

	// rootClient is a controller Client the Server can clone for
	// use by incoming connections.
	rootClient *comm.Client
	
	// rootBifrost is a Bifrost parser the Server can use for
	// incoming connections.
	rootBifrost comm.BifrostParser
}

// NewServer creates a new network server for a baps3d instance.
func NewServer(l *log.Logger, host string, rc *comm.Client, rb comm.BifrostParser) (*Server) {
	return &Server{
		l: l,
		host: host,
		rootClient: rc,
		rootBifrost: rb,
	}
}

func (s *Server) shutdownClient() {
	s.l.Println("shutting down")
	s.rootClient.Shutdown()
}

func (s *Server) Run() {
	defer s.shutdownClient()
	
	_, err := net.Listen("tcp", s.host)
	if err != nil {
		s.l.Println("couldn't open server:", err)
		return
	}
}
