package comm

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

type PlayerConn struct {
	net.Conn
	server *Server
}

func NewPlayerConn(conn net.Conn, s *Server) *PlayerConn {
	return &PlayerConn{ conn, s }
}

func (c *PlayerConn) Id() string {
	return c.RemoteAddr().String()
}

// TODO: return error via channel
func (c *PlayerConn) Loop() {
	scanner := bufio.NewScanner(c)
	OuterLoop:
	for {
		select {
			// check for server abort
			case <- c.server.abort:
				break OuterLoop

			default:
				if !scanner.Scan() {
					break OuterLoop
				}
				msgStr := scanner.Text()
				c.ReceivedString(msgStr)
		}	
	}
	
	if err := scanner.Err(); err != nil {
		c.server.error <- err
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
