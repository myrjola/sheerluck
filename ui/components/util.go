package components

import (
	"crypto/rand"
	"encoding/hex"
)

func shortUID() string {
	b := make([]byte, 4) //equals 8 characters
	rand.Read(b)
	return hex.EncodeToString(b)
}
