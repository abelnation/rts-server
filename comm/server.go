package comm

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	*net.TCPListener
	playerConn chan *PlayerConn
	playerMsg chan Message
	abort chan bool
	error chan error
	done chan bool
	playerConnWait *sync.WaitGroup
	isClosing bool
}

// Start new server
// @Overrides net.Listen
func Listen(laddr string) (*Server, error) {
	const netType string = "tcp"
	addr, err := net.ResolveTCPAddr(netType, laddr)
	if err != nil {
		return nil, err
	}
	
	tcpServer, err := net.ListenTCP(netType, addr)
	if err != nil {
		return nil, err
	}

	return &Server{ 
		tcpServer, 
		make(chan *PlayerConn), 
		make(chan Message),
		make(chan bool),
		make(chan error),
		make(chan bool),
		&sync.WaitGroup{},
		false,
	}, nil
}

func (s *Server) Loop() error {

	// set up os signal handlers for server abort
	go s.awaitSignals()

	// hook up player connection channel
	go s.awaitConnection()

	ServerLoop:
	for {

		select {
		case conn := <-s.playerConn:
			s.handlePlayerConn(conn)
			
		case msg := <-s.playerMsg:
			s.handleMessage(msg)	
			
		case err := <-s.error:
			s.handleError(err)

		case <-s.abort:
			fmt.Println("Server shutting down...")
			s.isClosing = true

			// wait for all connections to be closed
			go func() {
				fmt.Println("Waiting for client connections to finish...")
				s.playerConnWait.Wait()
				s.done <- true
			}()

			break ServerLoop	

		}
		
	}

	// Cleanup:
	// wait for all client connections to finish
	// meanwhile, log all errors that happen
	CleanupLoop:
	for {
		select {
			case msg := <-s.playerMsg:
				s.handleMessage(msg)
				
			case err := <-s.error:
				s.handleError(err)

			case <-s.done:
				break CleanupLoop
		}
	}

	fmt.Println("Done.")

	return nil
}

func (s *Server) handlePlayerConn(conn *PlayerConn) {
	fmt.Printf("Connected: %s\n", conn.Id())
	
	// increment wait queue
	s.playerConnWait.Add(1)
	
	// send welcome message
	conn.SendString("connected")

	// run PlayerConn Loop
	go func(conn *PlayerConn) {
		defer s.cleanupPlayerConn(conn)
		conn.Loop()
		fmt.Printf("Disconnected: %s\n", conn.Id())
	}(conn)
}

func (s *Server) handleMessage(msg Message) {
	fmt.Printf("%s: %s\n", msg.conn.Id(), msg.msg)
}

func (s *Server) handleError(err error) {
	fmt.Printf("error: %s\n", err)
}

func (s *Server) cleanupPlayerConn(conn *PlayerConn) {
	err := conn.Close()
	if err != nil {
		s.error <- err
	}
	// decrement wait queue
	s.playerConnWait.Done()
}

func (s *Server) awaitConnection() {
	for {

		// update deadline to 1s from now
        err := s.SetDeadline(time.Now().Add(time.Second))
        if err != nil {
        	s.error <- err
        	break
        }

		conn, err := s.Accept()
		// timeout allows us to check and see if server is closing
		// if so, stop accepting new requests
		if err != nil {

			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() && netErr.Temporary() {
				fmt.Println("taking a break.  checking for server close...")
				if (s.isClosing) {
					break
				} else {
					continue
				}
			}

			// unexpected error
			s.error <- err
			continue
		}
		// valid connection
		s.playerConn <- conn
	}
}

func (s *Server) awaitSignals() {
	// setup channel for os signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	signal := <- signals
	fmt.Printf("Sig received: %s\n", signal);
	close(s.abort)
}

//
// Listener overrides
//

func (s *Server) Accept() (*PlayerConn, error) {
	conn, err := s.TCPListener.Accept()
	if err != nil {
		return nil, err
	}

	c := NewPlayerConn(conn, s)
	return c, nil
}

func (s *Server) Close() error {
	// TODO: clean-up / wait on clean player conn close
	err := s.TCPListener.Close()
	if err != nil {
		return err
	}

	return nil
}

// 
// Private functions / methods
//
