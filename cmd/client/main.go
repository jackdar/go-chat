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
	username := flag.String("user", fmt.Sprintf("User%d", os.Getpid()), "Username")
	flag.Parse()

	log.Printf("Connecting to server at %s as %s", *addr, *username)

	c, err := client.NewClient(*addr, *username)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer c.Close()

	log.Println("Connected! Type '/help' for commands")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		if c.GetRoom() != "" {
			fmt.Printf("[%s] > ", c.GetRoom())
		} else {
			fmt.Print("> ")
		}

		if !scanner.Scan() {
			break
		}

		// Clear the input line immediately after Enter is pressed
		fmt.Print("\033[A\033[K")

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if err := handleCommand(c, line); err != nil {
			log.Printf("Error: %v", err)
		}
	}
}

func handleCommand(c *client.Client, line string) error {
	if !strings.HasPrefix(line, "/") {
		return c.SendMessage(line)
	}

	parts := strings.SplitN(line, " ", 2)
	cmd := parts[0]

	switch cmd {
	case "/help":
		fmt.Println("Commands:")
		fmt.Println("  /join <room-name>  - Join or create a room")
		fmt.Println("  /leave             - Leave current room")
		fmt.Println("  /help              - Show this help")
		fmt.Println("  /quit              - Exit")
		fmt.Println("  <message>          - Send a message to the current room")

	case "/join":
		if len(parts) < 2 {
			return fmt.Errorf("usage: /join <room-name>")
		}
		return c.JoinRoom(parts[1])

	case "/leave":
		return c.LeaveRoom()

	case "/quit", "/exit":
		os.Exit(0)

	default:
		return fmt.Errorf("unknown command: %s (type /help for commands)", cmd)
	}

	return nil
}
