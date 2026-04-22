package service

import (
	"context"
	"errors"
	"testing"

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/domain"
	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/repository"
)

// テストヘルパ: 共通のセットアップ（ユーザーと両者の vanilla 口座を用意）
func setup(t *testing.T) (context.Context, *Service, *domain.User, *domain.User, *domain.Account, *domain.Account) {
	t.Helper()
	ctx := context.Background()
	svc := New(repository.NewMemoryStore())
	a, err := svc.CreateUser(ctx, CreateUserInput{Name: "Alice", Email: "alice@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateUser(ctx, CreateUserInput{Name: "Bob", Email: "bob@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	aAcc, err := svc.OpenAccount(ctx, OpenAccountInput{UserID: a.ID, Flavor: domain.FlavorVanilla})
	if err != nil {
		t.Fatal(err)
	}
	bAcc, err := svc.OpenAccount(ctx, OpenAccountInput{UserID: b.ID, Flavor: domain.FlavorVanilla})
	if err != nil {
		t.Fatal(err)
	}
	return ctx, svc, a, b, aAcc, bAcc
}

func TestService_DepositAndWithdraw(t *testing.T) {
	ctx, svc, _, _, a, _ := setup(t)

	a2, err := svc.Deposit(ctx, a.ID, 10, "最初の一枚")
	if err != nil {
		t.Fatal(err)
	}
	if a2.Balance != 10 {
		t.Fatalf("want 10, got %d", a2.Balance)
	}

	a3, err := svc.Withdraw(ctx, a.ID, 3, "おやつ")
	if err != nil {
		t.Fatal(err)
	}
	if a3.Balance != 7 {
		t.Fatalf("want 7, got %d", a3.Balance)
	}

	// 取引履歴も2件残っている
	txs, err := svc.ListTransactions(ctx, a.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 2 {
		t.Fatalf("want 2 tx records, got %d", len(txs))
	}
}

func TestService_Withdraw_Insufficient_KeepsBalance(t *testing.T) {
	ctx, svc, _, _, a, _ := setup(t)
	_, _ = svc.Deposit(ctx, a.ID, 5, "")
	_, err := svc.Withdraw(ctx, a.ID, 10, "")
	if !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatalf("want ErrInsufficientBalance, got %v", err)
	}
	got, _ := svc.GetAccount(ctx, a.ID)
	if got.Balance != 5 {
		t.Fatalf("balance must be 5, got %d", got.Balance)
	}
	// 失敗取引は履歴に残さない
	txs, _ := svc.ListTransactions(ctx, a.ID, 0)
	if len(txs) != 1 {
		t.Fatalf("want 1 tx (only the deposit), got %d", len(txs))
	}
}

func TestService_Transfer_SameFlavor(t *testing.T) {
	ctx, svc, _, _, a, b := setup(t)
	_, _ = svc.Deposit(ctx, a.ID, 10, "")
	if err := svc.Transfer(ctx, TransferInput{
		FromAccountID: a.ID, ToAccountID: b.ID, Amount: 4, Memo: "プレゼント",
	}); err != nil {
		t.Fatal(err)
	}
	ga, _ := svc.GetAccount(ctx, a.ID)
	gb, _ := svc.GetAccount(ctx, b.ID)
	if ga.Balance != 6 || gb.Balance != 4 {
		t.Fatalf("balances wrong: a=%d b=%d", ga.Balance, gb.Balance)
	}
}

func TestService_Transfer_FlavorMismatch(t *testing.T) {
	ctx, svc, _, b, a, _ := setup(t)
	_, _ = svc.Deposit(ctx, a.ID, 10, "")
	// b の chocolate 口座を追加
	bChoco, err := svc.OpenAccount(ctx, OpenAccountInput{UserID: b.ID, Flavor: domain.FlavorChocolate})
	if err != nil {
		t.Fatal(err)
	}
	err = svc.Transfer(ctx, TransferInput{
		FromAccountID: a.ID, ToAccountID: bChoco.ID, Amount: 3,
	})
	if !errors.Is(err, domain.ErrFlavorMismatch) {
		t.Fatalf("want ErrFlavorMismatch, got %v", err)
	}
	// 両残高が変わらない
	ga, _ := svc.GetAccount(ctx, a.ID)
	gb, _ := svc.GetAccount(ctx, bChoco.ID)
	if ga.Balance != 10 || gb.Balance != 0 {
		t.Fatalf("balances changed: a=%d b=%d", ga.Balance, gb.Balance)
	}
}

func TestService_Transfer_Insufficient_Atomic(t *testing.T) {
	ctx, svc, _, _, a, b := setup(t)
	_, _ = svc.Deposit(ctx, a.ID, 2, "")
	err := svc.Transfer(ctx, TransferInput{
		FromAccountID: a.ID, ToAccountID: b.ID, Amount: 5,
	})
	if !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatalf("want ErrInsufficientBalance, got %v", err)
	}
	ga, _ := svc.GetAccount(ctx, a.ID)
	gb, _ := svc.GetAccount(ctx, b.ID)
	if ga.Balance != 2 || gb.Balance != 0 {
		t.Fatalf("rollback failed: a=%d b=%d", ga.Balance, gb.Balance)
	}
}

func TestService_Transfer_Self(t *testing.T) {
	ctx, svc, _, _, a, _ := setup(t)
	_, _ = svc.Deposit(ctx, a.ID, 5, "")
	err := svc.Transfer(ctx, TransferInput{FromAccountID: a.ID, ToAccountID: a.ID, Amount: 1})
	if !errors.Is(err, domain.ErrSelfTransfer) {
		t.Fatalf("want ErrSelfTransfer, got %v", err)
	}
}

func TestService_Exchange_WithinUser(t *testing.T) {
	ctx, svc, a, _, aV, _ := setup(t)
	aChoco, err := svc.OpenAccount(ctx, OpenAccountInput{UserID: a.ID, Flavor: domain.FlavorChocolate})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = svc.Deposit(ctx, aV.ID, 10, "")

	res, err := svc.Exchange(ctx, ExchangeInput{
		FromAccountID: aV.ID, ToAccountID: aChoco.ID, Amount: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.FromAmount != 10 || res.ToAmount != 8 {
		t.Fatalf("want (10,8), got (%d,%d)", res.FromAmount, res.ToAmount)
	}
	gV, _ := svc.GetAccount(ctx, aV.ID)
	gC, _ := svc.GetAccount(ctx, aChoco.ID)
	if gV.Balance != 0 || gC.Balance != 8 {
		t.Fatalf("balances wrong: V=%d C=%d", gV.Balance, gC.Balance)
	}
}

func TestService_Exchange_AcrossUsers_Rejected(t *testing.T) {
	ctx, svc, _, b, aV, _ := setup(t)
	bChoco, _ := svc.OpenAccount(ctx, OpenAccountInput{UserID: b.ID, Flavor: domain.FlavorChocolate})
	_, _ = svc.Deposit(ctx, aV.ID, 10, "")
	_, err := svc.Exchange(ctx, ExchangeInput{FromAccountID: aV.ID, ToAccountID: bChoco.ID, Amount: 5})
	if !errors.Is(err, domain.ErrForeignExchange) {
		t.Fatalf("want ErrForeignExchange, got %v", err)
	}
}
