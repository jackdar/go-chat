package server

import (
	"sync"
	"time"
)

type Room struct {
	Code string
	Name string
	Clients map[*Client]bool
	CreatedAt time.Time
	mu sync.RWMutex
}

func NewRoom(code string, name string) *Room {
	return &Room{
		Code: code,
		Name: name,
		Clients: make(map[*Client]bool),
		CreatedAt: time.Now(),
	}
}

func (r *Room) AddClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Clients[client] = true
	client.currentRoom = r.Code
}

func (r *Room) RemoveClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Clients, client)
	client.currentRoom = ""
}

func (r *Room) ClientCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.Clients)
}

func (r *Room) GetClients() []*Client {
	r.mu.Lock()
	defer r.mu.Unlock()

	clients := make([]*Client, 0, len(r.Clients))
	for client := range r.Clients {
		clients = append(clients, client)
	}
	return clients
}
