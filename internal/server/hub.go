package server

import (
	"log"
	"sync"
	"time"

	"github.com/jackdar/go-chat/internal/protocol"
)

type Hub struct {
	// connection -> Client
	clients map[*Client]bool

	// roomCode -> Room
	rooms map[string]*Room

	// username -> Client
	users map[string]*Client

	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	roomAction chan *RoomAction

	mu sync.RWMutex
}

type BroadcastMessage struct {
	roomCode string
	sender   *Client
	msgType  protocol.MessageType
	payload  any
}

type RoomAction struct {
	actionType RoomActionType
	client     *Client
	roomCode   string
	roomName   string
	response   chan *RoomActionResponse
}

type RoomActionType int

const (
	ActionCreateRoom RoomActionType = iota
	ActionJoinRoom
	ActionLeaveRoom
)

type RoomActionResponse struct {
	success  bool
	message  string
	roomCode string
	roomName string
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]*Room),
		users:      make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
		roomAction: make(chan *RoomAction),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			// - Add client to h.clients map
			h.clients[client] = true

			// - Add username to h.users map
			h.users[client.username] = client

			// - Log registration
			log.Printf("Client registered: %s", client.username)

		case client := <-h.unregister:
			// - Check if client exists
			if _, ok := h.clients[client]; !ok {
				log.Printf("Attempted to unregister non-existent client: %s", client.username)
			}

			// - Remove from current room (if any)
			// - Notify room members user left
			if client.currentRoom != "" {
				h.rooms[client.currentRoom].RemoveClient(client)
				h.broadcast <- &BroadcastMessage{
					roomCode: client.currentRoom,
					sender:   client,
					msgType:  protocol.MsgLeaveRoom,
					payload: protocol.ChatMessagePayload{
						Content: client.username + " has left the room.",
					},
				}
			}

			// - Remove from h.clients and h.users
			delete(h.clients, client)
			delete(h.users, client.username)

			// - Close client's send channel
			close(client.send)

		case msg := <-h.broadcast:
			// - Find room by msg.roomCode
			room, ok := h.rooms[msg.roomCode]
			if !ok {
				log.Printf("Broadcast to non-existent room: %s", msg.roomCode)
				continue
			}

			// - For each client in room:
			for client := range room.Clients {
				// - Encode message
				message, err := protocol.EncodeMessage(msg.msgType, msg.payload)
				if err != nil {
					log.Printf("Failed to encode broadcast message: %v", err)
					continue
				}

				select {
				// - Send to client's send channel (non-blocking)
				case client.send <- message:
					log.Printf("Broadcasted message to %s in room %s", client.username, msg.roomCode)

				//- If send fails, unregister client
				default:
					h.unregister <- client
				}
			}

		case action := <-h.roomAction:
			switch action.actionType {
			case ActionCreateRoom:
				// - Create room and add to h.rooms
				room := &Room{
					Code:      GenerateRoomCode(),
					Name:      action.roomName,
					Clients:   make(map[*Client]bool),
					CreatedAt: time.Now(),
					mu:        sync.RWMutex{},
				}

				// - Add room to hub
				h.rooms[room.Code] = room
				log.Printf("[RoomAction]ActionCreateRoom: Client %s created room %s", action.client.username, room.Code)

				// - Add client to room
				room.AddClient(action.client)
				log.Printf("[RoomAction]ActionCreateRoom: Client %s added to room %s", action.client.username, room.Code)

				// - Send success response
				action.response <- &RoomActionResponse{
					success:  true,
					message:  "Room created successfully",
					roomCode: room.Code,
					roomName: room.Name,
				}
				log.Printf("[RoomAction]ActionCreateRoom: Client %s has room %s", action.client.username, action.client.currentRoom)

				log.Printf("[RoomAction]ActionCreateRoom: Room has users:")
				for c := range room.Clients {
					log.Printf("  - %s", c.username)
				}

			case ActionJoinRoom:
				// - Check if room exists
				room, ok := h.rooms[action.roomCode]
				// - If not, send error response
				if !ok {
					log.Printf("[RoomAction]ActionJoinRoom: Room %s does not exist", action.roomCode)
					action.response <- &RoomActionResponse{
						success: false,
						message: "Room does not exist",
					}
					continue
				}

				// - Remove client from current room (if any)
				if action.client.currentRoom != "" {
					if r, ok := h.rooms[action.client.currentRoom]; ok {
						r.RemoveClient(action.client)
					} else {
						action.client.currentRoom = ""
					}
				}

				// - Add client to new room
				room.AddClient(action.client)
				log.Printf("[RoomAction]ActionJoinRoom: Client %s joined room %s", action.client.username, action.roomCode)

				// - Notify all room members
				h.broadcast <- &BroadcastMessage{
					roomCode: action.roomCode,
					sender:   action.client,
					msgType:  protocol.MsgUserJoined,
					payload: protocol.UserEventPayload{
						Username: action.client.username,
						RoomCode: action.roomCode,
					},
				}

				// - Send success response
				action.response <- &RoomActionResponse{
					success:  true,
					message:  "Joined room successfully",
					roomCode: action.roomCode,
					roomName: action.roomName,
				}

			case ActionLeaveRoom:
				// - Find client's current room
				currentRoom := action.client.currentRoom
				if currentRoom == "" {
					log.Printf("[RoomAction]ActionLeaveRoom: Client not in any room")
					action.response <- &RoomActionResponse{
						success: false,
						message: "Client not in any room",
					}
					continue
				}

				// - Remove client from room
				if r, ok := h.rooms[currentRoom]; ok {
					r.RemoveClient(action.client)
					log.Printf("[RoomAction]ActionLeaveRoom: Client %s left room %s", action.client.username, currentRoom)

					// - Notify remaining members
					h.broadcast <- &BroadcastMessage{
						roomCode: currentRoom,
						sender:   action.client,
						msgType:  protocol.MsgUserLeft,
						payload: protocol.UserEventPayload{
							Username: action.client.username,
							RoomCode: currentRoom,
						},
					}

					// - If room empty, delete room
					if r.ClientCount() == 0 {
						delete(h.rooms, currentRoom)
						log.Printf("[RoomAction]ActionLeaveRoom: Deleted empty room %s", currentRoom)
					}

					// - Send success response
					action.response <- &RoomActionResponse{
						success: true,
						message: "Left room successfully",
					}
				} else {
					// Room not found, send error
					action.response <- &RoomActionResponse{
						success: false,
						message: "Room does not exist",
					}
				}
			}
		}
	}
}
