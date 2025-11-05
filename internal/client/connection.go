package client

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/jackdar/go-chat/internal/protocol"
)

type Connection struct {
	conn     net.Conn
	username string
	roomCode string
	roomName string
	resp     chan *protocol.Message
	mu       sync.Mutex
	Updates  chan string
}

func NewConnection(address, username string) (*Connection, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	c := &Connection{
		conn:     conn,
		username: username,
		resp:     make(chan *protocol.Message, 1),
		Updates:  make(chan string, 1),
	}

	go c.ReadMessages()

	if err := c.authenticate(); err != nil {
		conn.Close()
		return nil, err
	}

	log.Printf("Authenticated with server as %s", username)

	return c, nil
}

func (c *Connection) write(msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.conn.Write(msg)
	return err
}

func (c *Connection) authenticate() error {
	payload := protocol.AuthPayload{
		Username: c.username,
		Profile:  "Go Chat User",
	}

	msg, err := protocol.EncodeMessage(protocol.MsgAuth, payload)
	if err != nil {
		return fmt.Errorf("failed to encode auth message: %w", err)
	}

	err = c.write(msg)
	return err
}

func (c *Connection) CreateRoom(roomName string) (string, error) {
	payload := protocol.CreateRoomPayload{
		RoomName: roomName,
	}

	msg, err := protocol.EncodeMessage(protocol.MsgCreateRoom, payload)
	if err != nil {
		return "", err
	}

	if err := c.write(msg); err != nil {
		return "", err
	}

	select {
	case response := <-c.resp:
		var respPayload protocol.ServerResponsePayload
		if err := protocol.DecodePayload(response, &respPayload); err != nil {
			return "", fmt.Errorf("failed to decode response: %w", err)
		}
		if !respPayload.Success {
			return "", fmt.Errorf("server error: %s", respPayload.Message)
		}
		roomCode, ok := respPayload.Data["room_code"].(string)
		if !ok {
			return "", fmt.Errorf("no room code in response")
		}
		c.roomCode = roomCode
		return roomCode, nil

	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("timeout waiting for server response")
	}
}

func (c *Connection) JoinRoom(roomCode string) error {
	payload := protocol.JoinRoomPayload{
		RoomCode: roomCode,
	}

	msg, err := protocol.EncodeMessage(protocol.MsgJoinRoom, payload)
	if err != nil {
		return err
	}

	if err := c.write(msg); err != nil {
		return err
	}

	select {
	case response := <-c.resp:
		var respPayload protocol.ServerResponsePayload

		if err := protocol.DecodePayload(response, &respPayload); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		if !respPayload.Success {
			return fmt.Errorf("server error: %s", respPayload.Message)
		}

		roomCode, ok := respPayload.Data["room_code"].(string)
		if !ok {
			return fmt.Errorf("no room code in response")
		}
		c.roomCode = roomCode

		roomName, ok := respPayload.Data["room_name"].(string)
		if !ok {
			return fmt.Errorf("no room name in response")
		}
		c.roomName = roomName

		return nil

	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for server response")
	}
}

func (c *Connection) LeaveRoom() error {
	if c.roomCode == "" {
		return fmt.Errorf("not in any room")
	}

	msg, err := protocol.EncodeMessage(protocol.MsgLeaveRoom, nil)
	if err != nil {
		return err
	}

	err = c.write(msg)
	c.roomCode = ""
	return err
}

func (c *Connection) SendMessage(content string) error {
	if c.roomCode == "" {
		return fmt.Errorf("not in a room. Use '/create' or '/join' first")
	}

	payload := protocol.ChatMessagePayload{
		Content: content,
	}

	msg, err := protocol.EncodeMessage(protocol.MsgChatMessage, payload)
	if err != nil {
		return err
	}

	err = c.write(msg)
	return err
}

func (c *Connection) ReadMessages() {
	for {
		msg, err := protocol.DecodeMessage(c.conn)
		if err != nil {
			log.Printf("Connection closed: %v", err)
			return
		}

		if msg.Type == protocol.MsgServerResponse {
			select {
			case c.resp <- msg:
			case c.Updates <- c.roomName:
			default:
				c.handleMessage(msg)
			}
			continue
		}

		c.handleMessage(msg)
	}
}

func (c *Connection) handleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgChatMessage:
		var payload protocol.ChatMessagePayload
		if err := protocol.DecodePayload(msg, &payload); err != nil {
			log.Printf("Failed to decode chat message: %v", err)
			return
		}
		fmt.Printf("\r💬 %s\n> ", payload.Content)

	case protocol.MsgUserJoined:
		var payload protocol.UserEventPayload
		if err := protocol.DecodePayload(msg, &payload); err != nil {
			log.Printf("Failed to decode user joined: %v", err)
			return
		}
		if payload.Username != c.username {
			fmt.Printf("\r✅ %s joined the room\n> ", payload.Username)
		}

	case protocol.MsgUserLeft:
		var payload protocol.UserEventPayload
		if err := protocol.DecodePayload(msg, &payload); err != nil {
			log.Printf("Failed to decode user left: %v", err)
			return
		}
		fmt.Printf("\r❌ %s left the room\n> ", payload.Username)

	case protocol.MsgServerResponse:
		var payload protocol.ServerResponsePayload
		if err := protocol.DecodePayload(msg, &payload); err != nil {
			log.Printf("Failed to decode server response: %v", err)
			return
		}
		if payload.Success {
			fmt.Printf("\r✓ %s\n> ", payload.Message)
		} else {
			fmt.Printf("\r✗ Error: %s\n> ", payload.Message)
		}

	default:
		log.Printf("Unknown message type: %d", msg.Type)
	}
}

func (c *Connection) Close() error {
	return c.conn.Close()
}

func (c *Connection) GetUsername() string {
	return c.username
}

func (c *Connection) GetRoomCode() string {
	return c.roomCode
}

func (c *Connection) GetRoomName() string {
	return c.roomName
}
