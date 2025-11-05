package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackdar/go-chat/internal/client"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "Server address")
	username := flag.String("user", "User"+fmt.Sprint(os.Getpid()), "Username")
	flag.Parse()

	log.Printf("Connecting to server at %s as %s", *addr, *username)

	conn, err := client.NewConnection(*addr, *username)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	log.Println("Connected! Type '/help' for commands")

	inputCh := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			inputCh <- scanner.Text()
		}
		close(inputCh)
	}()

	for {
		// print prompt once before waiting (optional)
		if conn.GetRoomName() != "" {
			fmt.Printf("[%s] > ", conn.GetRoomName())
		} else {
			fmt.Print("> ")
		}

		select {
		case line, ok := <-inputCh:
			if !ok {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if err := handleCommand(conn, line); err != nil {
				log.Printf("Error: %v", err)
			}

		case newRoom := <-conn.Updates:
			// redraw prompt on room change
			if newRoom != "" {
				fmt.Printf("\r[%s] > ", newRoom)
			} else {
				fmt.Print("\r> ")
			}
		}
	}
}

func handleCommand(conn *client.Connection, line string) error {
	parts := strings.SplitN(line, " ", 2)
	cmd := parts[0]

	switch cmd {
	case "/help":
		fmt.Println("Commands:")
		fmt.Println("  /create <room-name>  - Create a new chat room")
		fmt.Println("  /join <room-code>    - Join an existing room")
		fmt.Println("  /help                - Help")
		fmt.Println("  /quit                - Exit")
		fmt.Println("  <message>            - Send a message to the current room")

	case "/create":
		if len(parts) < 2 {
			return fmt.Errorf("usage: create <room-name>")
		}
		roomCode, err := conn.CreateRoom(parts[1])
		if err != nil {
			return err
		}
		fmt.Printf("Room created with code: %s\n", roomCode)

	case "/join":
		if len(parts) < 2 {
			return fmt.Errorf("usage: join <room-code>")
		}
		if err := conn.JoinRoom(parts[1]); err != nil {
			return err
		}
		fmt.Printf("Joining room %s...\n", parts[1])

	case "quit", "exit":
		os.Exit(0)

	default:
		return conn.SendMessage(line)
	}

	return nil
}
