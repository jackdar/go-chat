package server

import (
	"fmt"
	"log"
	"net"

	"github.com/jackdar/go-chat/internal/protocol"
)

type Client struct {
	hub         *Hub
	conn        net.Conn
	send        chan []byte
	username    string
	currentRoom *Room
}

func NewClient(hub *Hub, conn net.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
}

func (c *Client) authenticate() error {
	msg, err := protocol.DecodeMessage(c.conn)
	if err != nil {
		return fmt.Errorf("failed to read auth message: %w", err)
	}

	if msg.Type != protocol.MsgAuth {
		return fmt.Errorf("expected auth message, got %d", msg.Type)
	}

	var authPayload protocol.AuthPayload
	if err := protocol.DecodePayload(msg, &authPayload); err != nil {
		return fmt.Errorf("failed to decode auth payload: %w", err)
	}

	c.username = authPayload.Username
	return nil
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	if err := c.authenticate(); err != nil {
		log.Printf("Authentication failed: %v", err)
		return
	}

	c.hub.register <- c

	for {

		for {
			msg, err := protocol.DecodeMessage(c.conn)
			if err != nil {
				log.Printf("Failed to read message: %v", err)
				return
			}

			c.handleMessage(msg)

		}
	}
}

func (c *Client) handleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgCreateRoom:
		var payload protocol.CreateRoomPayload
		protocol.DecodePayload(msg, &payload)

		responseCh := make(chan *RoomActionResponse, 1)

		c.hub.roomAction <- &RoomAction{
			actionType: ActionCreateRoom,
			client:     c,
			response:   responseCh,
		}

		resp := <-responseCh

		log.Printf("Client %s created room %s", c.username, payload.RoomName)

		c.sendResponse(resp)

	case protocol.MsgJoinRoom:
		var payload protocol.JoinRoomPayload
		protocol.DecodePayload(msg, &payload)

		responseCh := make(chan *RoomActionResponse, 1)
		c.hub.roomAction <- &RoomAction{
			actionType: ActionJoinRoom,
			client:     c,
			roomCode:   payload.RoomCode,
			response:   responseCh,
		}

		log.Printf("Client %s joined room %s", c.username, payload.RoomCode)

		resp := <-responseCh
		c.sendResponse(resp)

	case protocol.MsgChatMessage:
		var payload protocol.ChatMessagePayload
		protocol.DecodePayload(msg, &payload)

		if c.currentRoom == nil {
			log.Printf("Client %s tried to send message without joining a room", c.username)
			return
		}

		c.hub.broadcast <- &BroadcastMessage{
			roomCode: c.currentRoom.Code,
			sender:   c,
			msgType:  protocol.MsgChatMessage,
			payload: protocol.ChatMessagePayload{
				Content: fmt.Sprintf("%s: %s", c.username, payload.Content),
			},
		}

		log.Printf("[%s](%s) > %s", c.hub.rooms[c.currentRoom].Name, c.username, payload.Content)

	default:
		log.Printf("Unknown message type: %d", msg.Type)
	}
}

func (c *Client) sendResponse(resp *RoomActionResponse) {
	payload := protocol.ServerResponsePayload{
		Success: resp.success,
		Message: resp.message,
		Data:    make(map[string]any),
	}

	if resp.roomCode != "" {
		payload.Data["room_code"] = resp.roomCode
	}

	if resp.roomName != "" {
		payload.Data["room_name"] = resp.roomName
	}

	msg, err := protocol.EncodeMessage(protocol.MsgServerResponse, payload)
	if err != nil {
		log.Printf("Failed to encode server response: %v", err)
		return
	}

	c.send <- msg
}

func (c *Client) WritePump() {
	defer c.conn.Close()

	for message := range c.send {
		if _, err := c.conn.Write(message); err != nil {
			log.Printf("Failed to write message to %s: %v", c.username, err)
			return
		}
	}
}
