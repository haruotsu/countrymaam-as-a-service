package domain

import (
	"testing"
	"time"
)

func TestNewSessionToken_UniqueAndLongEnough(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		tok, err := NewSessionToken()
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if len(tok) < 32 {
			t.Fatalf("token too short: %d chars", len(tok))
		}
		if seen[tok] {
			t.Fatalf("duplicate token produced: %s", tok)
		}
		seen[tok] = true
	}
}

func TestSession_IsExpired(t *testing.T) {
	past := time.Now().Add(-time.Second)
	future := time.Now().Add(time.Hour)

	expired := &Session{ExpiresAt: past}
	active := &Session{ExpiresAt: future}

	if !expired.IsExpired(time.Now()) {
		t.Fatal("past session should be expired")
	}
	if active.IsExpired(time.Now()) {
		t.Fatal("future session should not be expired")
	}
}
