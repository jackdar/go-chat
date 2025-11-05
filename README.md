# Go Chat

A lightweight, real-time TCP chat server and client written in Go, featuring a simplified action channel pattern and newline-delimited JSON protocol.

## Features

- **Real-time messaging** - TCP-based chat with instant message delivery
- **Multi-room support** - Create and join multiple chat rooms dynamically
- **Simple protocol** - Human-readable newline-delimited JSON messages
- **Composable architecture** - Unified action channel pattern for easy extensibility
- **Concurrent** - Handles multiple clients efficiently with goroutines

## Quick Start

### Prerequisites

- Go 1.16 or higher
- Make (optional, for convenience commands)

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd go-chat

# Build binaries
make all
```

### Running the Server

```bash
# Run with default settings (localhost:8080)
make run

# Or with custom host and port
go run ./cmd/server/main.go -host localhost -port 9000
```

### Running the Client

```bash
# Run with default settings (connects to localhost:8080, auto-generated username)
go run ./cmd/client/main.go

# Or with custom settings
go run ./cmd/client/main.go -addr localhost:8080 -user Alice
```

## Client Commands

Once connected, use these commands to interact with the chat:

- `/join <room-name>` - Join or create a chat room
- `/leave` - Leave your current room
- `/help` - Display available commands
- `/quit` - Disconnect and exit

Any other text will be sent as a chat message to your current room.

## Project Structure

```
go-chat/
├── cmd/
│   ├── server/        # Server entry point
│   └── client/        # Client entry point
├── internal/
│   ├── server/        # Server implementation
│   │   ├── server.go      # TCP listener
│   │   ├── hub.go         # Central coordinator
│   │   ├── connection.go  # Client connection handler
│   │   └── room.go        # Chat room management
│   ├── client/        # Client implementation
│   │   └── client.go      # Client logic
│   └── protocol/      # Protocol definitions
│       ├── message.go     # Message types
│       └── encoding.go    # JSON encoding/decoding
├── bin/               # Compiled binaries
├── Makefile          # Build automation
└── README.md         # This file
```

## Architecture

### Core Design Principles

**Unified Action Channel Pattern**: Instead of multiple separate channels, the system uses a single `actions` channel that accepts a discriminated union (Action struct with ActionType). This makes the system more composable and easier to extend.

**Newline-Delimited JSON Protocol**: Messages are simple JSON objects terminated by newlines, making them human-readable and easy to debug with tools like `netcat` or `telnet`.

**Direct Room Names**: Rooms are identified by their names directly with no indirection or complexity.

### Components

**Server** (`internal/server/`)
- `Server`: TCP listener that accepts connections and spawns goroutines per client
- `Hub`: Central coordinator running in its own goroutine, processes all actions from a single channel
- `Connection`: Represents a connected client with dedicated goroutines for reading and writing
- `Room`: Thread-safe chat room with mutex-protected connection management

**Client** (`internal/client/`)
- `Client`: Handles TCP connection, authentication, and message I/O
- Simple API: `JoinRoom(name)`, `LeaveRoom()`, `SendMessage(content)`
- Background goroutine for asynchronous message reading

**Protocol** (`internal/protocol/`)
- Flat `Message` struct with fields: `type`, `username`, `room`, `content`, `success`, `error`
- Message types: `auth`, `join`, `leave`, `chat`, `system`
- Encoding: Newline-delimited JSON (`json.Marshal` + `\n`)

### Hub Action Pattern

The Hub processes 5 action types through a single channel:

```go
type Action struct {
    Type     ActionType          // Register, Unregister, Join, Leave, Broadcast
    Conn     *Connection         // Which connection
    Room     string              // Room name (for join operations)
    Message  *protocol.Message   // Message to broadcast
    Response chan error          // Optional response for synchronous operations
}
```

All state changes flow through: `hub.actions <- Action{...}`

### Control Flow

1. **Connection Lifecycle**: Accept → Authenticate → Register → Process messages → Unregister on disconnect
2. **Joining Rooms**: Connection sends Join action → Hub creates room if needed → Adds connection → Broadcasts join event
3. **Broadcasting**: Connection sends Broadcast action → Hub distributes to all connections in room
4. **Room Cleanup**: When the last user leaves, Hub automatically deletes the room

## Protocol Details

Messages are JSON objects terminated by newlines (`\n`). Example:

```json
{"type":"auth","username":"Alice"}
{"type":"join","room":"general"}
{"type":"chat","username":"Alice","room":"general","content":"Hello, world!"}
{"type":"leave","room":"general"}
```

### Message Types

- `auth` - Initial authentication with username
- `join` - Join a room (creates if doesn't exist)
- `leave` - Leave current room
- `chat` - Send message to room
- `system` - Server notifications (join/leave announcements)

## Development

### Building

```bash
# Build both binaries
make all

# Build server only
go build -o bin/server ./cmd/server/main.go

# Build client only
go build -o bin/client ./cmd/client/main.go
```

### Testing

```bash
go test ./...
```

### Manual Testing

You can test the server with `netcat`:

```bash
# Connect to server
nc localhost 8080

# Send authentication
{"type":"auth","username":"TestUser"}

# Join a room
{"type":"join","room":"test"}

# Send a message
{"type":"chat","content":"Hello!"}
```

## Docker Support

A Dockerfile is included for containerized deployment:

```bash
# Build image
docker build -t go-chat .

# Run server
docker run -p 8080:8080 go-chat
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
