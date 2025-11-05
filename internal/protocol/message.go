package protocol

import "encoding/json"

type MessageType uint8

const (
	MsgAuth MessageType = iota + 1
	MsgCreateRoom
	MsgJoinRoom
	MsgLeaveRoom
	MsgChatMessage
	MsgServerResponse
	MsgRoomList
	MsgUserJoined
	MsgUserLeft
)

type Message struct {
	Type MessageType `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type AuthPayload struct {
	Username string `json:"username"`
}

type CreateRoomPayload struct {
	RoomName string `json:"room_name"`
}

type JoinRoomPayload struct {
	RoomCode string `json:"room_code"`
}

type ChatMessagePayload struct {
	Content string `json:"content"`
}

type ServerResponsePayload struct {
	Success bool `json:"success"`
	Message string `json:"message"`
	Data map[string]any `json:"data,omitempty"`
}

type UserEventPayload struct {
	Username string `json:"username"`
	RoomCode string `json:"room_code"`
}
