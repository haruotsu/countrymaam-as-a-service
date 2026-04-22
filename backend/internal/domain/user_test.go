package domain

import (
	"errors"
	"testing"
)

func TestNewUser_OK(t *testing.T) {
	u, err := NewUser("  Haruto  ", "Foo@Example.com")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if u.Name != "Haruto" {
		t.Fatalf("name trim failed: %q", u.Name)
	}
	if u.Email != "foo@example.com" {
		t.Fatalf("email normalize failed: %q", u.Email)
	}
}

func TestNewUser_Invalid(t *testing.T) {
	cases := []struct{ name, email string }{
		{"", "x@y.z"},
		{"n", ""},
		{"n", "noatmark"},
	}
	for _, c := range cases {
		if _, err := NewUser(c.name, c.email); !errors.Is(err, ErrInvalidUser) {
			t.Fatalf("want ErrInvalidUser for (%q,%q), got %v", c.name, c.email, err)
		}
	}
}
