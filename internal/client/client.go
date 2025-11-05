package client

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/jackdar/go-chat/internal/protocol"
)

type Client struct {
	conn     net.Conn
	username string
	room     string
	mu       sync.Mutex
}

func NewClient(address, username string) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	c := &Client{
		conn:     conn,
		username: username,
	}

	// Authenticate
	if err := c.authenticate(); err != nil {
		conn.Close()
		return nil, err
	}

	// Start reading messages in background
	go c.readMessages()

	log.Printf("Connected as %s", username)

	return c, nil
}

func (c *Client) authenticate() error {
	msg := &protocol.Message{
		Type:     protocol.TypeAuth,
		Username: c.username,
	}

	return protocol.WriteMessage(c.conn, msg)
}

func (c *Client) JoinRoom(roomName string) error {
	msg := &protocol.Message{
		Type: protocol.TypeJoin,
		Room: roomName,
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := protocol.WriteMessage(c.conn, msg); err != nil {
		return err
	}

	c.room = roomName
	return nil
}

func (c *Client) LeaveRoom() error {
	msg := &protocol.Message{
		Type: protocol.TypeLeave,
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := protocol.WriteMessage(c.conn, msg); err != nil {
		return err
	}

	c.room = ""
	return nil
}

func (c *Client) SendMessage(content string) error {
	c.mu.Lock()
	room := c.room
	c.mu.Unlock()

	if room == "" {
		return fmt.Errorf("not in a room. Use '/join <room-name>' first")
	}

	msg := &protocol.Message{
		Type:    protocol.TypeChat,
		Content: content,
	}

	return protocol.WriteMessage(c.conn, msg)
}

func (c *Client) readMessages() {
	reader := bufio.NewReader(c.conn)

	for {
		msg, err := protocol.DecodeMessage(reader)
		if err != nil {
			log.Printf("Connection closed: %v", err)
			return
		}

		c.handleMessage(msg)
	}
}

func (c *Client) handleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.TypeChat:
		fmt.Printf("\r💬 %s: %s\n> ", msg.Username, msg.Content)

	case protocol.TypeSystem:
		if msg.Error != "" {
			fmt.Printf("\r❌ Error: %s\n> ", msg.Error)
		} else if msg.Content != "" {
			if msg.Username != "" && msg.Username != c.username {
				// User joined/left message
				fmt.Printf("\r✨ %s\n> ", msg.Content)
			} else if msg.Success {
				// Success message (e.g., joined room)
				fmt.Printf("\r✅ %s\n> ", msg.Content)
			} else {
				fmt.Printf("\r📢 %s\n> ", msg.Content)
			}
		}

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetRoom() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.room
}
