package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func Code() string {
	var b [18]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Errorf("secure random: %w", err))
	}
	return hex.EncodeToString(b[:])
}
func UUID() string {
	s := Code()
	return fmt.Sprintf("%s-%s-4%s-a%s-%s", s[:8], s[8:12], s[13:16], s[17:20], s[20:32])
}
