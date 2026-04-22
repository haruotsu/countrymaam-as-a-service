package domain

import (
	"errors"
	"testing"
)

func TestNewAccount_InitialBalanceIsZero(t *testing.T) {
	a := NewAccount("u1", FlavorVanilla)
	if a.Balance != 0 {
		t.Fatalf("new account balance must be 0, got %d", a.Balance)
	}
	if a.UserID != "u1" {
		t.Fatalf("user id not set")
	}
	if a.Flavor != FlavorVanilla {
		t.Fatalf("flavor not set")
	}
}

func TestAccount_Deposit(t *testing.T) {
	a := NewAccount("u1", FlavorVanilla)
	if err := a.Deposit(5); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if a.Balance != 5 {
		t.Fatalf("want 5, got %d", a.Balance)
	}
	if err := a.Deposit(3); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if a.Balance != 8 {
		t.Fatalf("want 8, got %d", a.Balance)
	}
}

func TestAccount_Deposit_Invalid(t *testing.T) {
	a := NewAccount("u1", FlavorVanilla)
	for _, n := range []int64{0, -1, -100} {
		if err := a.Deposit(n); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("amount %d: want ErrInvalidAmount, got %v", n, err)
		}
	}
	if a.Balance != 0 {
		t.Fatalf("balance must stay 0 after invalid deposits, got %d", a.Balance)
	}
}

func TestAccount_Withdraw(t *testing.T) {
	a := NewAccount("u1", FlavorVanilla)
	_ = a.Deposit(10)
	if err := a.Withdraw(3); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if a.Balance != 7 {
		t.Fatalf("want 7, got %d", a.Balance)
	}
}

func TestAccount_Withdraw_Insufficient(t *testing.T) {
	a := NewAccount("u1", FlavorVanilla)
	_ = a.Deposit(10)
	err := a.Withdraw(11)
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("want ErrInsufficientBalance, got %v", err)
	}
	if a.Balance != 10 {
		t.Fatalf("balance must be unchanged on failed withdraw, got %d", a.Balance)
	}
}

func TestAccount_Withdraw_Invalid(t *testing.T) {
	a := NewAccount("u1", FlavorVanilla)
	_ = a.Deposit(10)
	for _, n := range []int64{0, -1} {
		if err := a.Withdraw(n); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("amount %d: want ErrInvalidAmount, got %v", n, err)
		}
	}
}
