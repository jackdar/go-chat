package server

import (
	"regexp"
	"testing"
)

func TestGenerateRoomCode(t *testing.T) {
	pattern := regexp.MustCompile(`^[A-Z0-9]{6}$`)

	code := GenerateRoomCode()
	if !pattern.MatchString(code) {
		t.Errorf("GenerateRoomCode() = %q; want format [A-Z0-9]{6}", code)
	}

	if len(code) != 6 {
		t.Errorf("GenerateRoomCode() length = %d; want 6", len(code))
	}
}

func TestGenerateRoomCodeUniqueness(t *testing.T) {
	codes := make(map[string]bool)
	iterations := 1000

	for range iterations {
		code := GenerateRoomCode()
		codes[code] = true
	}

	if len(codes) < iterations-10 {
		t.Errorf("Generated %d unique codes out of %d; too many collisions",
			len(codes), iterations)
	}
}
