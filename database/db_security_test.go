package database

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestDefaultPasswordHashIsNotStoredAsPlaintext(t *testing.T) {
	hash, err := defaultPasswordHash()
	if err != nil {
		t.Fatalf("defaultPasswordHash() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("default password hash = %q, want bcrypt format", hash)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(defaultPassword)); err != nil {
		t.Fatalf("default password hash does not verify: %v", err)
	}
	if hash == defaultPassword {
		t.Fatal("default password was returned in plaintext")
	}
}
