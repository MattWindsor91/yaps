package netsrv

import (
	"log"
	"net"
	"sync"

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

	// wg is a WaitGroup that tracks all inner server goroutines.
	// The server main loop won't terminate until the WaitGroup hits zero.
	wg sync.WaitGroup
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
func New(l *log.Logger, host string, rc *comm.Client, rb comm.BifrostParser) *Server {
	return &Server{
		l:           l,
		host:        host,
		rootClient:  rc,
		rootBifrost: rb,
		clients:     make(map[client]struct{}),
	}
}

func (s *Server) shutdownController() {
	s.l.Println("shutting down")
	s.rootClient.Shutdown()
}

// newClient sets up the server s to handle incoming connection c.
func (s *Server) newClient(c net.Conn) {
	s.l.Println("new connection:", c.RemoteAddr().String())

	cli := client{
		conn: c,
	}

	s.clients[cli] = struct{}{}
}

// hangUpAllClients gracefully closes all connected clients on s.
func (s *Server) hangUpAllClients() {
	for c := range s.clients {
		s.hangUpClient(&c)
	}
}

// hangUpClient closes the client pointed to by c.
func (s *Server) hangUpClient(c *client) {
	s.l.Println("hanging up:", c.conn.RemoteAddr().String())
	c.Close()
	delete(s.clients, *c)
}

// Run runs the net server main loop.
func (s *Server) Run() {
	defer s.wg.Wait()
	defer s.shutdownController()

	ln, err := net.Listen("tcp", s.host)
	if err != nil {
		s.l.Println("couldn't open server:", err)
		return
	}

	connCh := make(chan net.Conn)
	cerrCh := make(chan error)

	defer func() {
		s.hangUpAllClients()
		if err := ln.Close(); err != nil {
			s.l.Println("error closing listener:", err)
		}
		s.l.Println("closed listener")

		// The acceptor is going to send us a 'my listener has closed!'
		// error, then close the channel.  This means we'll have to
		// pretend to listen for that error first.
		for _ = range cerrCh {
		}
	}()

	s.l.Println("now listening on", s.host)
	s.wg.Add(1)
	go func() {
		s.acceptClients(ln, connCh, cerrCh)
		s.wg.Done()
	}()

	for {
		select {
		case err := <-cerrCh:
			s.l.Println("error accepting connections:", err)
			return
		case conn := <-connCh:
			s.newClient(conn)
		case <-s.rootClient.Done:
			s.l.Println("received controller shutdown")
			return
		}
	}
}

// acceptClients keeps spinning, accepting clients on ln and sending them to
// connCh, until ln closes.
// It then sends the error on errCh and closes both channels.
func (s *Server) acceptClients(ln net.Listener, connCh chan<- net.Conn, cerrCh chan<- error) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			cerrCh <- err
			close(cerrCh)
			return
		}
		select {
		case connCh <- conn:
		case <-s.rootClient.Done:
			// The main loop will have closed, so won't be listening
			// for connections, but we're waiting for it to close our acceptor.
			// TODO(@MattWindsor91): necessary?
			conn.Close()
		}
	}
}
