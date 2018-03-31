package netsrv

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

	// clients is a map containing all connected clients.
	clients map[client]struct{}
}

// client holds the server-side state of a baps3d TCP client.
type client struct {
	// conn holds the client socket.
	conn net.Conn
	
	// buf holds the client buffer.
	buf [4096]byte
}

// Close closes the given client.
func (c *client) Close() {
	c.conn.Close()
}

// New creates a new network server for a baps3d instance.
func New(l *log.Logger, host string, rc *comm.Client, rb comm.BifrostParser) (*Server) {
	return &Server{
		l: l,
		host: host,
		rootClient: rc,
		rootBifrost: rb,
		clients: make(map[client]struct{}),
	}
}

func (s *Server) shutdownController() {
	s.l.Println("shutting down")
	s.rootClient.Shutdown()
}

// newClient sets up the server s to handle incoming connection c.
func (s *Server) newClient(c net.Conn) {
	s.l.Println("new connection:", c.RemoteAddr().String())

	client := client{
		conn: c,
	}

	s.clients[client] = struct{}{}
}

// hangUpAllClients gracefully closes all connected clients on s.
func (s *Server) hangUpAllClients() {
	for c, _ := range s.clients {
		s.hangUpClient(&c)
	}
}

// hangUpClient closes the client pointed to by c.
func (s *Server) hangUpClient(c *client) {
	c.Close()
	delete(s.clients, *c)
}
	
func (s *Server) Run() {
	defer s.shutdownController()
	
	ln, err := net.Listen("tcp", s.host)
	if err != nil {
		s.l.Println("couldn't open server:", err)
		return
	}

	defer ln.Close()
	s.l.Println("now listening on", s.host)

	connCh := make(chan net.Conn)
	cerrCh := make(chan error)
	
	go acceptClients(ln, connCh, cerrCh)

	for {
		select {
		case err := <-cerrCh:
			s.l.Println("error accepting connections:", err)
			return
		case conn := <-connCh:
			s.newClient(conn)
		case <-s.rootClient.Done:
			s.l.Println("received controller shutdown")
			s.hangUpAllClients()
			return

		}
	}
}

// acceptClients keeps spinning, accepting clients on ln and sending them to
// connCh, until ln closes.
// It then sends the error on errCh.
func acceptClients(ln net.Listener, connCh chan<- net.Conn, cerrCh chan<- error) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			cerrCh <- err
			return
		}
		connCh <- conn
	}	
}
