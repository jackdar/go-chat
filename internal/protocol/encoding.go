package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

func EncodeMessage(msgType MessageType, payload any) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	msg := Message{
		Type:    msgType,
		Payload: payloadBytes,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal message: %w", err)
	}

	length := uint32(len(msgBytes))
	result := make([]byte, 4+len(msgBytes))
	binary.BigEndian.PutUint32(result[0:4], length)
	copy(result[4:], msgBytes)

	// log.Printf("DEBUG: length=%d raw=%q", length, string(msgBytes))

	return result, nil
}

func DecodeMessage(reader io.Reader) (*Message, error) {
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(reader, lengthBuf); err != nil {
		return nil, fmt.Errorf("read length: %w", err)
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length > 1024*1024 {
		return nil, fmt.Errorf("message too large: %d bytes", length)
	}

	msgBuf := make([]byte, length)
	if _, err := io.ReadFull(reader, msgBuf); err != nil {
		return nil, fmt.Errorf("read message: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(msgBuf, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}

	return &msg, nil
}

func DecodePayload(msg *Message, target any) error {
	return json.Unmarshal(msg.Payload, target)
}
