package comm

import (
 	"bufio"
	"fmt"
	"net"
	"strings"
//	"time"
)

type PlayerConn struct {
	*net.TCPConn
	server *Server
	connMsg chan string
	done chan bool
}

func NewPlayerConn(conn *net.TCPConn, s *Server) *PlayerConn {
	return &PlayerConn{ 
		conn, 
		s,
		make(chan string),
		make(chan bool, 1),
	}
}

func (c *PlayerConn) Id() string {
	return c.RemoteAddr().String()
}

func (c *PlayerConn) Loop() {
	
	go c.awaitMessages()
	// scanner := NewTCPConnScanner(c)

	OuterLoop:
	for {
		select {
			// check for server abort
			case <-c.server.abort:
				fmt.Println("PlayerConn detected server abort.  Shutting down...")
				break OuterLoop

			case msg := <-c.connMsg:
				c.ReceivedString(msg)
		}	
	}
	
}

func (c *PlayerConn) awaitMessages() {
	scanner := bufio.NewScanner(c)
	for scanner.Scan() {
		c.connMsg <- scanner.Text()	
	}

	if err := scanner.Err(); err != nil {
		c.server.error <- fmt.Errorf("player conn scan err: %v", err)
	}
}

func (c *PlayerConn) ReceivedString(msg string) {
	c.ReceivedMessage(Message{ strings.TrimSpace(msg), c })
}

func (c *PlayerConn) ReceivedMessage(msg Message) {
	c.server.playerMsg <- msg
}

func (c *PlayerConn) SendString(msg string) {
	c.SendMessage(Message{ strings.TrimSpace(msg), c })
}

func (c *PlayerConn) SendMessage(msg Message) {
	// TODO: properly marshall messages
	strMsg := fmt.Sprintf("server: %s\n", msg.msg)
	c.Write([]byte(strMsg))
}

//
// IPConn overrides
//
