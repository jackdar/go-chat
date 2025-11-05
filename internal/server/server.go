package server

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type Server struct {
	config   *Config
	hub      *Hub
	listener net.Listener
	wg       sync.WaitGroup
	done     chan struct{}
}

func NewServer(config *Config) *Server {
	return &Server{
		config: config,
		hub:    NewHub(),
		done:   make(chan struct{}),
	}
}

func (s *Server) newListener() (net.Listener) {
	listener, err := net.Listen("tcp", s.config.Address())
	if err != nil {
		log.Fatalf("Failed to listen %w", err)
	}

	log.Printf("Server listening on %s", s.config.Address())

	return listener
}

func (s *Server) Start() error {
	go s.hub.Run()

	s.listener = s.newListener()

	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

func (s *Server) Stop() error {
	close(s.done)

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return fmt.Errorf("Error closing listener: %w", err)
		}
	}

	s.wg.Wait()

	return nil
}

func (s *Server) acceptConnections() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				log.Println("Stopped accepting connections")
				return
			default:
				log.Printf("Error accepting connection: %v", err)
				continue
			}
		}

		log.Printf("New connection from %s", conn.RemoteAddr())

		s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	client := NewClient(s.hub, conn)

	s.hub.register <- client

	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		client.ReadPump()
	}()
	go func() {
		defer s.wg.Done()
		client.WritePump()
	}()
}
