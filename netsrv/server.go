package netsrv

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/UniversityRadioYork/bifrost-go"

	"github.com/UniversityRadioYork/baps3d/controller"
)

// Server holds the internal state of a baps3d TCP server.
type Server struct {
	// log is the Server's logger.
	log *log.Logger

	// host is the Server's host:port string.
	host string

	// rootClient is a controller Client the Server can clone for
	// use by incoming connections.
	rootClient *controller.Client

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
func New(l *log.Logger, host string, rc *controller.Client) *Server {
	return &Server{
		log:          l,
		host:         host,
		rootClient:   rc,
		accConn:      make(chan net.Conn),
		accErr:       make(chan error),
		clientHangUp: make(chan *Client),
		clientErr:    make(chan error),
		done:         make(chan struct{}),
		clients:      make(map[Client]struct{}),
	}
}

func (s *Server) shutdownController(ctx context.Context) {
	s.log.Println("shutting down")
	if err := s.rootClient.Shutdown(ctx); err != nil {
		s.log.Println("couldn't shut down gracefully:", err)
	}
}

// newConnection sets up the server s to handle incoming connection c.
// It does not close c on error.
func (s *Server) newConnection(ctx context.Context, c net.Conn) error {
	cname := c.RemoteAddr().String()
	s.log.Println("new connection:", cname)

	conClient, err := s.rootClient.Copy(ctx)
	if err != nil {
		return err
	}

	conBifrost, conBifrostClient, err := conClient.Bifrost(ctx)
	if err != nil {
		return err
	}

	ioClient := bifrost.IoClient{
		Conn:    c,
		Bifrost: conBifrostClient,
	}

	cli := Client{
		name:      cname,
		ioClient:  &ioClient,
		conClient: conClient,
		log:       s.log,
	}

	s.clients[cli] = struct{}{}

	s.wg.Add(1)
	go func() {
		cli.Run(ctx, conBifrost, s.clientHangUp)
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
func (s *Server) Run(ctx context.Context) {
	defer s.wg.Wait()
	defer s.shutdownController(ctx)

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

	s.mainLoop(ctx)

	close(s.done)
	s.hangUpAllClients()
	if err := ln.Close(); err != nil {
		s.log.Println("error closing listener:", err)
	}
	s.log.Println("closed listener")
}

// mainLoop is the server's main connection handling loop.
func (s *Server) mainLoop(ctx context.Context) {
	done := ctx.Done()
	for {
		select {
		case err := <-s.accErr:
			s.log.Println("error accepting connections:", err)
			return
		case conn := <-s.accConn:
			cname := conn.RemoteAddr().String()
			if err := s.newConnection(ctx, conn); err != nil {
				s.log.Printf("error registering connection %s: %s\n", cname, err.Error())
				if cerr := conn.Close(); err != nil {
					s.log.Printf("further error closing connection %s: %s\n", cname, cerr.Error())
				}
			}
		case c := <-s.clientHangUp:
			s.hangUpClient(c)
		case <-s.rootClient.Rx:
			// Drain any messages sent to the root client.
		case <-done:
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
