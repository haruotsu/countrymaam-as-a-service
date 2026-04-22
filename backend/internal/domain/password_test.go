package domain

import (
	"errors"
	"testing"
)

func TestHashPassword_OK(t *testing.T) {
	h, err := HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if h == "" || h == "correct-horse" {
		t.Fatalf("hash must be non-empty and not equal to the password, got %q", h)
	}
}

func TestHashPassword_TooShort(t *testing.T) {
	if _, err := HashPassword("short"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("want ErrWeakPassword, got %v", err)
	}
}

func TestVerifyPassword(t *testing.T) {
	h, _ := HashPassword("correct-horse-battery")
	if err := VerifyPassword(h, "correct-horse-battery"); err != nil {
		t.Fatalf("want match, got %v", err)
	}
	if err := VerifyPassword(h, "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}
