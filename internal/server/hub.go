package server

import (
	"fmt"
	"log"

	"github.com/jackdar/go-chat/internal/protocol"
)

// ActionType defines the type of action to perform
type ActionType int

const (
	ActionRegister ActionType = iota
	ActionUnregister
	ActionJoin
	ActionLeave
	ActionBroadcast
)

// Action represents a unified action sent to the Hub
type Action struct {
	Type     ActionType
	Conn     *Connection
	Room     string
	Message  *protocol.Message
	Response chan error // Optional response channel for synchronous operations
}

// Hub manages all connections and rooms through a single action channel
type Hub struct {
	actions chan Action

	// State (only accessed from Run goroutine)
	connections map[*Connection]bool
	rooms       map[string]*Room
	users       map[string]*Connection
}

func NewHub() *Hub {
	return &Hub{
		actions:     make(chan Action, 256),
		connections: make(map[*Connection]bool),
		rooms:       make(map[string]*Room),
		users:       make(map[string]*Connection),
	}
}

func (h *Hub) Run() {
	for action := range h.actions {
		switch action.Type {
		case ActionRegister:
			h.handleRegister(action)
		case ActionUnregister:
			h.handleUnregister(action)
		case ActionJoin:
			h.handleJoin(action)
		case ActionLeave:
			h.handleLeave(action)
		case ActionBroadcast:
			h.handleBroadcast(action)
		}
	}
}

func (h *Hub) handleRegister(action Action) {
	h.connections[action.Conn] = true
	h.users[action.Conn.username] = action.Conn
	log.Printf("Registered: %s", action.Conn.username)

	if action.Response != nil {
		action.Response <- nil
	}
}

func (h *Hub) handleUnregister(action Action) {
	if _, ok := h.connections[action.Conn]; !ok {
		log.Printf("Attempted to unregister non-existent connection: %s", action.Conn.username)
		return
	}

	// Leave current room if in one
	if action.Conn.currentRoom != nil {
		h.leaveRoom(action.Conn)
	}

	delete(h.connections, action.Conn)
	delete(h.users, action.Conn.username)
	close(action.Conn.send)

	log.Printf("Unregistered: %s", action.Conn.username)
}

func (h *Hub) handleJoin(action Action) {
	var err error

	// Leave current room if in one
	if action.Conn.currentRoom != nil {
		h.leaveRoom(action.Conn)
	}

	// Get or create room
	room, exists := h.rooms[action.Room]
	if !exists {
		room = NewRoom(action.Room)
		h.rooms[action.Room] = room
		log.Printf("Created room: %s", action.Room)
	}

	// Add connection to room
	room.AddConnection(action.Conn)
	log.Printf("%s joined room: %s", action.Conn.username, action.Room)

	// Broadcast join event to room
	h.broadcastToRoom(room.Name, &protocol.Message{
		Type:     protocol.TypeSystem,
		Username: action.Conn.username,
		Content:  fmt.Sprintf("%s joined the room", action.Conn.username),
	}, nil)

	// Send success response to the connection
	action.Conn.send <- &protocol.Message{
		Type:    protocol.TypeSystem,
		Success: true,
		Room:    room.Name,
		Content: fmt.Sprintf("Joined room: %s", room.Name),
	}

	if action.Response != nil {
		action.Response <- err
	}
}

func (h *Hub) handleLeave(action Action) {
	if action.Conn.currentRoom == nil {
		if action.Response != nil {
			action.Response <- fmt.Errorf("not in any room")
		}
		return
	}

	h.leaveRoom(action.Conn)

	if action.Response != nil {
		action.Response <- nil
	}
}

func (h *Hub) handleBroadcast(action Action) {
	if action.Conn.currentRoom == nil {
		log.Printf("%s tried to broadcast without being in a room", action.Conn.username)
		return
	}

	h.broadcastToRoom(action.Conn.currentRoom.Name, action.Message, action.Conn)
}

// Helper methods

func (h *Hub) leaveRoom(conn *Connection) {
	if conn.currentRoom == nil {
		return
	}

	room := conn.currentRoom
	roomName := room.Name
	room.RemoveConnection(conn)

	log.Printf("%s left room: %s", conn.username, roomName)

	// Broadcast leave event
	h.broadcastToRoom(roomName, &protocol.Message{
		Type:     protocol.TypeSystem,
		Username: conn.username,
		Content:  fmt.Sprintf("%s left the room", conn.username),
	}, nil)

	// Delete room if empty
	if room.ConnectionCount() == 0 {
		delete(h.rooms, roomName)
		log.Printf("Deleted empty room: %s", roomName)
	}
}

func (h *Hub) broadcastToRoom(roomName string, msg *protocol.Message, excludeConn *Connection) {
	room, ok := h.rooms[roomName]
	if !ok {
		log.Printf("Broadcast to non-existent room: %s", roomName)
		return
	}

	for conn := range room.Connections {
		// Skip the excluded connection (usually the sender)
		if conn == excludeConn {
			continue
		}

		select {
		case conn.send <- msg:
		default:
			// Connection's send buffer is full, unregister it
			log.Printf("Failed to send to %s, unregistering", conn.username)
			h.actions <- Action{Type: ActionUnregister, Conn: conn}
		}
	}
}
