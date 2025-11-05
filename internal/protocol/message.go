package protocol

// Message represents all communication between client and server
// Simple flat structure - only relevant fields are populated per message type
type Message struct {
	Type     string `json:"type"`               // "auth", "join", "leave", "chat", "system"
	Username string `json:"username,omitempty"` // For auth and system messages
	Room     string `json:"room,omitempty"`     // Room name (no codes)
	Content  string `json:"content,omitempty"`  // Chat content or system message
	Success  bool   `json:"success,omitempty"`  // For server responses
	Error    string `json:"error,omitempty"`    // Error message if any
}

// Message type constants
const (
	TypeAuth   = "auth"
	TypeJoin   = "join"
	TypeLeave  = "leave"
	TypeChat   = "chat"
	TypeSystem = "system"
)
