package service

import (
	"context"
	"errors"
	"testing"

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/domain"
	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/repository"
)

// テストヘルパ: 共通のセットアップ（ユーザー2人と各自の vanilla 口座を用意）
func setup(t *testing.T) (context.Context, *Service, *domain.User, *domain.User, *domain.Account, *domain.Account) {
	t.Helper()
	ctx := context.Background()
	svc := New(repository.NewMemoryStore())
	a, _, err := svc.Register(ctx, RegisterInput{Name: "Alice", Email: "alice@example.com", Password: "password-a"})
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := svc.Register(ctx, RegisterInput{Name: "Bob", Email: "bob@example.com", Password: "password-b"})
	if err != nil {
		t.Fatal(err)
	}
	aAcc, err := svc.OpenAccount(ctx, a.ID, domain.FlavorVanilla)
	if err != nil {
		t.Fatal(err)
	}
	bAcc, err := svc.OpenAccount(ctx, b.ID, domain.FlavorVanilla)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, svc, a, b, aAcc, bAcc
}

func TestService_DepositAndWithdraw(t *testing.T) {
	ctx, svc, a, _, aAcc, _ := setup(t)

	got, err := svc.Deposit(ctx, a.ID, aAcc.ID, 10, "最初の一枚")
	if err != nil {
		t.Fatal(err)
	}
	if got.Balance != 10 {
		t.Fatalf("want 10, got %d", got.Balance)
	}

	got, err = svc.Withdraw(ctx, a.ID, aAcc.ID, 3, "おやつ")
	if err != nil {
		t.Fatal(err)
	}
	if got.Balance != 7 {
		t.Fatalf("want 7, got %d", got.Balance)
	}
}

func TestService_Deposit_ByNonOwner_IsForbidden(t *testing.T) {
	ctx, svc, _, b, aAcc, _ := setup(t)
	// Bob が Alice の口座に入金しようとする
	_, err := svc.Deposit(ctx, b.ID, aAcc.ID, 5, "")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

func TestService_Withdraw_ByNonOwner_IsForbidden(t *testing.T) {
	ctx, svc, a, b, aAcc, _ := setup(t)
	_, _ = svc.Deposit(ctx, a.ID, aAcc.ID, 5, "")
	_, err := svc.Withdraw(ctx, b.ID, aAcc.ID, 3, "")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

func TestService_GetMyAccount_OtherUser_IsForbidden(t *testing.T) {
	ctx, svc, _, b, aAcc, _ := setup(t)
	_, err := svc.GetMyAccount(ctx, b.ID, aAcc.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

func TestService_Withdraw_Insufficient_KeepsBalance(t *testing.T) {
	ctx, svc, a, _, aAcc, _ := setup(t)
	_, _ = svc.Deposit(ctx, a.ID, aAcc.ID, 5, "")
	_, err := svc.Withdraw(ctx, a.ID, aAcc.ID, 10, "")
	if !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatalf("want ErrInsufficientBalance, got %v", err)
	}
	got, _ := svc.GetMyAccount(ctx, a.ID, aAcc.ID)
	if got.Balance != 5 {
		t.Fatalf("want 5, got %d", got.Balance)
	}
}

func TestService_Transfer_SameFlavor(t *testing.T) {
	ctx, svc, a, _, aAcc, bAcc := setup(t)
	_, _ = svc.Deposit(ctx, a.ID, aAcc.ID, 10, "")
	if err := svc.Transfer(ctx, TransferInput{
		ViewerID: a.ID, FromAccountID: aAcc.ID, ToAccountID: bAcc.ID, Amount: 4,
	}); err != nil {
		t.Fatal(err)
	}
	ga, _ := svc.GetMyAccount(ctx, a.ID, aAcc.ID)
	if ga.Balance != 6 {
		t.Fatalf("a balance %d", ga.Balance)
	}
}

func TestService_Transfer_NonOwner_Forbidden(t *testing.T) {
	ctx, svc, a, b, aAcc, bAcc := setup(t)
	_, _ = svc.Deposit(ctx, a.ID, aAcc.ID, 10, "")
	// b が a の口座から抜いて自分に送ろうとする
	err := svc.Transfer(ctx, TransferInput{
		ViewerID: b.ID, FromAccountID: aAcc.ID, ToAccountID: bAcc.ID, Amount: 1,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

func TestService_Transfer_Insufficient_Atomic(t *testing.T) {
	ctx, svc, a, _, aAcc, bAcc := setup(t)
	_, _ = svc.Deposit(ctx, a.ID, aAcc.ID, 2, "")
	err := svc.Transfer(ctx, TransferInput{
		ViewerID: a.ID, FromAccountID: aAcc.ID, ToAccountID: bAcc.ID, Amount: 5,
	})
	if !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatalf("want ErrInsufficientBalance, got %v", err)
	}
	ga, _ := svc.GetMyAccount(ctx, a.ID, aAcc.ID)
	if ga.Balance != 2 {
		t.Fatalf("rollback failed: %d", ga.Balance)
	}
}

func TestService_Exchange_WithinUser(t *testing.T) {
	ctx, svc, a, _, aV, _ := setup(t)
	aChoco, err := svc.OpenAccount(ctx, a.ID, domain.FlavorChocolate)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = svc.Deposit(ctx, a.ID, aV.ID, 10, "")
	res, err := svc.Exchange(ctx, ExchangeInput{
		ViewerID: a.ID, FromAccountID: aV.ID, ToAccountID: aChoco.ID, Amount: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ToAmount != 8 {
		t.Fatalf("want 8, got %d", res.ToAmount)
	}
}

func TestService_Exchange_AcrossUsers_Rejected(t *testing.T) {
	ctx, svc, a, b, aV, _ := setup(t)
	bChoco, _ := svc.OpenAccount(ctx, b.ID, domain.FlavorChocolate)
	_, _ = svc.Deposit(ctx, a.ID, aV.ID, 10, "")
	// a が他人の口座を指定 → 所有権 Forbidden
	_, err := svc.Exchange(ctx, ExchangeInput{
		ViewerID: a.ID, FromAccountID: aV.ID, ToAccountID: bChoco.ID, Amount: 5,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

func TestService_RegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	svc := New(repository.NewMemoryStore())
	u, sess, err := svc.Register(ctx, RegisterInput{Name: "A", Email: "a@example.com", Password: "password-1"})
	if err != nil {
		t.Fatal(err)
	}
	if sess.Token == "" {
		t.Fatal("session token should be issued at register")
	}
	// Authenticate でトークンから戻せる
	got, err := svc.Authenticate(ctx, sess.Token)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID {
		t.Fatalf("user id mismatch")
	}

	// Login
	_, sess2, err := svc.Login(ctx, LoginInput{Email: "a@example.com", Password: "password-1"})
	if err != nil {
		t.Fatal(err)
	}
	if sess2.Token == sess.Token {
		t.Fatal("a new token should be issued per login")
	}

	// 誤ったパスワード
	_, _, err = svc.Login(ctx, LoginInput{Email: "a@example.com", Password: "wrong"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
	// 存在しないメール
	_, _, err = svc.Login(ctx, LoginInput{Email: "nobody@example.com", Password: "x"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}

	// Logout
	if err := svc.Logout(ctx, sess.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(ctx, sess.Token); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated, got %v", err)
	}
}

func TestService_Register_WeakPassword(t *testing.T) {
	ctx := context.Background()
	svc := New(repository.NewMemoryStore())
	_, _, err := svc.Register(ctx, RegisterInput{Name: "A", Email: "a@example.com", Password: "short"})
	if !errors.Is(err, domain.ErrWeakPassword) {
		t.Fatalf("want ErrWeakPassword, got %v", err)
	}
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	ctx := context.Background()
	svc := New(repository.NewMemoryStore())
	_, _, _ = svc.Register(ctx, RegisterInput{Name: "A", Email: "a@example.com", Password: "password-1"})
	_, _, err := svc.Register(ctx, RegisterInput{Name: "B", Email: "A@Example.com", Password: "password-2"})
	if !errors.Is(err, domain.ErrDuplicateUserEmail) {
		t.Fatalf("want ErrDuplicateUserEmail, got %v", err)
	}
}
