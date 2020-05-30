package main

import (
	"crypto/sha1"
	"encoding/hex"
)

func hasher(password string) string {
	// Hash password
	hasher := sha1.New()
	hasher.Write([]byte(password))
	passwordHashHex := hasher.Sum(nil)

	passwordHash := hex.EncodeToString(passwordHashHex)
	return passwordHash
}
