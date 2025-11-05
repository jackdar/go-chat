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

func (s *Server) Start() error {
	go s.hub.Run()

	listener, err := net.Listen("tcp", s.config.Address())
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = listener

	log.Printf("Server listening on %s", s.config.Address())

	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

func (s *Server) Stop() error {
	close(s.done)

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return fmt.Errorf("error closing listener: %w", err)
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

		connection := NewConnection(s.hub, conn)
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			connection.Run()
		}()
	}
}
