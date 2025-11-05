package server

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/jackdar/go-chat/internal/protocol"
)

type Connection struct {
	hub         *Hub
	conn        net.Conn
	send        chan *protocol.Message
	username    string
	currentRoom *Room
}

func NewConnection(hub *Hub, conn net.Conn) *Connection {
	return &Connection{
		hub:  hub,
		conn: conn,
		send: make(chan *protocol.Message, 256),
	}
}

// Run handles both reading and writing in a single goroutine
func (c *Connection) Run() {
	defer func() {
		c.hub.actions <- Action{Type: ActionUnregister, Conn: c}
		c.conn.Close()
	}()

	// Authenticate first
	if err := c.authenticate(); err != nil {
		log.Printf("Authentication failed: %v", err)
		return
	}

	// Register with hub
	c.hub.actions <- Action{Type: ActionRegister, Conn: c}

	// Create reader
	reader := bufio.NewReader(c.conn)

	// Start write goroutine
	done := make(chan struct{})
	go c.writePump(done)

	// Read loop
	for {
		msg, err := protocol.DecodeMessage(reader)
		if err != nil {
			log.Printf("Failed to read message from %s: %v", c.username, err)
			close(done)
			return
		}
		c.handleMessage(msg)
	}
}

func (c *Connection) authenticate() error {
	reader := bufio.NewReader(c.conn)
	msg, err := protocol.DecodeMessage(reader)
	if err != nil {
		return fmt.Errorf("failed to read auth message: %w", err)
	}

	if msg.Type != protocol.TypeAuth {
		return fmt.Errorf("expected auth message, got %s", msg.Type)
	}

	if msg.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	c.username = msg.Username
	log.Printf("Authenticated: %s", c.username)
	return nil
}

func (c *Connection) handleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.TypeJoin:
		if msg.Room == "" {
			c.sendError("room name cannot be empty")
			return
		}
		c.hub.actions <- Action{
			Type: ActionJoin,
			Conn: c,
			Room: msg.Room,
		}

	case protocol.TypeLeave:
		c.hub.actions <- Action{
			Type: ActionLeave,
			Conn: c,
		}

	case protocol.TypeChat:
		if msg.Content == "" {
			return
		}
		// Broadcast to others
		c.hub.actions <- Action{
			Type: ActionBroadcast,
			Conn: c,
			Message: &protocol.Message{
				Type:     protocol.TypeChat,
				Username: c.username,
				Content:  msg.Content,
			},
		}
		// Send local echo to sender
		c.send <- &protocol.Message{
			Type:     protocol.TypeChat,
			Username: c.username,
			Content:  msg.Content,
		}

	default:
		log.Printf("Unknown message type from %s: %s", c.username, msg.Type)
	}
}

func (c *Connection) sendError(errMsg string) {
	c.send <- &protocol.Message{
		Type:    protocol.TypeSystem,
		Success: false,
		Error:   errMsg,
	}
}

func (c *Connection) writePump(done chan struct{}) {
	defer c.conn.Close()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := protocol.WriteMessage(c.conn, msg); err != nil {
				log.Printf("Failed to write message to %s: %v", c.username, err)
				return
			}
		case <-done:
			return
		}
	}
}
