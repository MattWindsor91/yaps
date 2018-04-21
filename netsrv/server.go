package netsrv

import (
	"log"
	"net"
	"sync"

	"github.com/UniversityRadioYork/baps3d/comm"
)

// Server holds the internal state of a baps3d TCP server.
type Server struct {
	// log is the Server's logger.
	log *log.Logger

	// host is the Server's host:port string.
	host string

	// rootClient is a controller Client the Server can clone for
	// use by incoming connections.
	rootClient *comm.Client

	// rootBifrost is a Bifrost parser the Server can use for
	// incoming connections.
	rootBifrost comm.BifrostParser

	// clients is a map containing all connected clients.
	clients map[Client]struct{}

	// accConn is a channel used by the acceptor goroutine to send new
	// connections to the main goroutine.
	accConn chan net.Conn

	// accErr is a channel used by the acceptor goroutine to send errors
	// to the main goroutine.
	// Errors landing from accErr are considered fatal.
	accErr chan error

	// clientHangUp is a channel used by client goroutines to send
	// disconnections to the main goroutine.
	// It sends a pointer to the client to disconnect.
	clientHangUp chan *Client

	// clientErr is a channel used by client goroutines to send
	// errors to the main goroutine.
	// The client will send a hangup request if the error is fatal.
	clientErr chan error

	// done is a channel closed when the main loop terminates.
	// This is used to signal all goroutines to close, if they haven't
	// already.
	done chan struct{}

	// wg is a WaitGroup that tracks all inner server goroutines.
	// The server main loop won't terminate until the WaitGroup hits zero.
	wg sync.WaitGroup
}

// New creates a new network server for a baps3d instance.
func New(l *log.Logger, host string, rc *comm.Client, rb comm.BifrostParser) *Server {
	return &Server{
		log:          l,
		host:         host,
		rootClient:   rc,
		rootBifrost:  rb,
		accConn:      make(chan net.Conn),
		accErr:       make(chan error),
		clientHangUp: make(chan *Client),
		clientErr:    make(chan error),
		done:         make(chan struct{}),
		clients:      make(map[Client]struct{}),
	}
}

func (s *Server) shutdownController() {
	s.log.Println("shutting down")
	s.rootClient.Shutdown()
}

// newConnection sets up the server s to handle incoming connection c.
func (s *Server) newConnection(c net.Conn) error {
	cname := c.RemoteAddr().String()
	s.log.Println("new connection:", cname)

	conClient, err := s.rootClient.Copy()
	if err != nil {
		_ = c.Close()
		return err
	}
	conBifrost, conBifrostClient := comm.NewBifrost(conClient, s.rootBifrost)
	cli := Client{
		name:       cname,
		conn:       c,
		conClient:  conClient,
		conBifrost: conBifrostClient,
		log:        s.log,
	}

	s.clients[cli] = struct{}{}

	s.wg.Add(3)
	go func() {
		cli.RunTx()
		// Only hang up if the server is still around.
		// Otherwise, we'll just hang here waiting for the server to answer,
		// while the server hangs up the client anyway.
		select {
		case s.clientHangUp <- &cli:
		case <-s.done:
		}
		s.wg.Done()
	}()
	go func() {
		cli.RunRx()
		s.wg.Done()
	}()
	go func() {
		conBifrost.Run()
		s.wg.Done()
	}()

	return nil
}

// hangUpAllClients gracefully closes all connected clients on s.
func (s *Server) hangUpAllClients() {
	for c := range s.clients {
		s.hangUpClient(&c)
	}
}

// hangUpClient closes the client pointed to by c.
func (s *Server) hangUpClient(c *Client) {
	s.log.Println("hanging up:", c.name)
	if err := c.Close(); err != nil {
		s.log.Printf("couldn't gracefully close %s: %s\n", c.name, err.Error())
	}
	delete(s.clients, *c)
}

// Run prepares and runs the net server main loop.
func (s *Server) Run() {
	defer s.wg.Wait()
	defer s.shutdownController()

	ln, err := net.Listen("tcp", s.host)
	if err != nil {
		s.log.Println("couldn't open server:", err)
		return
	}

	s.log.Println("now listening on", s.host)
	s.wg.Add(1)
	go func() {
		s.acceptClients(ln)
		s.wg.Done()
	}()

	s.mainLoop()

	close(s.done)
	s.hangUpAllClients()
	if err := ln.Close(); err != nil {
		s.log.Println("error closing listener:", err)
	}
	s.log.Println("closed listener")
}

// mainLoop is the server's main connection handling loop.
func (s *Server) mainLoop() {
	for {
		select {
		case err := <-s.accErr:
			s.log.Println("error accepting connections:", err)
			return
		case conn := <-s.accConn:
			cname := conn.RemoteAddr().String()
			if err := s.newConnection(conn); err != nil {
				s.log.Printf("error registering connection %s: %s\n", cname, err.Error())
			}
		case c := <-s.clientHangUp:
			s.hangUpClient(c)
		case <-s.rootClient.Rx:
			// Drain any messages sent to the root client.
		case <-s.rootClient.Done:
			s.log.Println("received controller shutdown")
			return
		}
	}
}

// acceptClients keeps spinning, accepting clients on ln and sending them to
// connCh, until ln closes.
// It then sends the error on errCh and closes both channels.
func (s *Server) acceptClients(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			// Only send the error if the main loop is listening
			select {
			case s.accErr <- err:
			case <-s.done:
			}
			close(s.accErr)
			close(s.accConn)
			return
		}

		// Only forward connections if the main loop actually wants them
		select {
		case s.accConn <- conn:
		case <-s.done:
			// TODO(@MattWindsor91): necessary?
			_ = conn.Close()
		}
	}
}
