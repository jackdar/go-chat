package server

import (
	"crypto/rand"
	"math/big"
)

func GenerateRoomCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		index, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[index.Int64()]
	}
	return string(code)
}
