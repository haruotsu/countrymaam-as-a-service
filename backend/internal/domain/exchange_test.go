package domain

import (
	"errors"
	"testing"
)

func TestExchange_Identity(t *testing.T) {
	got, err := Exchange(FlavorVanilla, FlavorVanilla, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 10 {
		t.Fatalf("same flavor should be identity: got %d", got)
	}
}

func TestExchange_VanillaToChocolate(t *testing.T) {
	// floor(10 * 1.0 / 1.2) = 8
	got, err := Exchange(FlavorVanilla, FlavorChocolate, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 8 {
		t.Fatalf("want 8, got %d", got)
	}
}

func TestExchange_MatchaToVanilla(t *testing.T) {
	// floor(6 * 1.5 / 1.0) = 9
	got, err := Exchange(FlavorMatcha, FlavorVanilla, 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 9 {
		t.Fatalf("want 9, got %d", got)
	}
}

func TestExchange_ZeroAmount(t *testing.T) {
	_, err := Exchange(FlavorVanilla, FlavorChocolate, 0)
	if !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("want ErrInvalidAmount, got %v", err)
	}
}

func TestExchange_NegativeAmount(t *testing.T) {
	_, err := Exchange(FlavorVanilla, FlavorChocolate, -5)
	if !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("want ErrInvalidAmount, got %v", err)
	}
}

func TestExchange_TooSmall(t *testing.T) {
	// floor(1 * 1.0 / 1.5) = 0 → "少なすぎて換えられない"
	_, err := Exchange(FlavorVanilla, FlavorMatcha, 1)
	if !errors.Is(err, ErrExchangeTooSmall) {
		t.Fatalf("want ErrExchangeTooSmall, got %v", err)
	}
}
