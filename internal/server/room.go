package server

import (
	"sync"
	"time"
)

type Room struct {
	Name        string
	Connections map[*Connection]bool
	CreatedAt   time.Time
	mu          sync.RWMutex
}

func NewRoom(name string) *Room {
	return &Room{
		Name:        name,
		Connections: make(map[*Connection]bool),
		CreatedAt:   time.Now(),
	}
}

func (r *Room) AddConnection(connection *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Connections[connection] = true
	connection.currentRoom = r
}

func (r *Room) RemoveConnection(connection *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Connections, connection)
	connection.currentRoom = nil
}

func (r *Room) ConnectionCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Connections)
}

func (r *Room) GetConnections() []*Connection {
	r.mu.RLock()
	defer r.mu.RUnlock()

	connections := make([]*Connection, 0, len(r.Connections))
	for connection := range r.Connections {
		connections = append(connections, connection)
	}
	return connections
}
