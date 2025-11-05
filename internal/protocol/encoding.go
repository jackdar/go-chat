package protocol

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// EncodeMessage encodes a message as newline-delimited JSON
func EncodeMessage(msg *Message) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal message: %w", err)
	}
	// Append newline for line-delimited protocol
	return append(data, '\n'), nil
}

// DecodeMessage reads and decodes a newline-delimited JSON message
func DecodeMessage(reader io.Reader) (*Message, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max message size

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read message: %w", err)
		}
		return nil, io.EOF
	}

	var msg Message
	if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}

	return &msg, nil
}

// WriteMessage encodes and writes a message
func WriteMessage(writer io.Writer, msg *Message) error {
	data, err := EncodeMessage(msg)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}
