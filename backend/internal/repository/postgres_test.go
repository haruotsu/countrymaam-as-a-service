package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/domain"
)

// TEST_DATABASE_URL が設定されている環境でのみ走る統合テスト。
// docker compose up -d db した後に
//   TEST_DATABASE_URL=postgres://cmaas:cmaas@localhost:5432/cmaas?sslmode=disable go test ./internal/repository/
// で実行する想定。
func setupStore(t *testing.T) (*pgStore, context.Context) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping DB integration tests")
	}
	ctx := context.Background()
	store, closeFn, err := NewStoreFromURL(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(closeFn)
	pg := store.(*pgStore)
	if err := pg.CleanAll(ctx); err != nil {
		t.Fatalf("clean: %v", err)
	}
	return pg, ctx
}

func TestPG_UserCRUD(t *testing.T) {
	store, ctx := setupStore(t)
	u, _ := domain.NewUser("Alice", "alice@example.com")
	if err := store.Users().Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	if u.ID == "" {
		t.Fatal("id should be populated")
	}
	got, err := store.Users().FindByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "alice@example.com" {
		t.Fatalf("got %+v", got)
	}
}

func TestPG_User_DuplicateEmail(t *testing.T) {
	store, ctx := setupStore(t)
	u1, _ := domain.NewUser("A", "dup@example.com")
	u2, _ := domain.NewUser("B", "dup@example.com")
	if err := store.Users().Create(ctx, u1); err != nil {
		t.Fatal(err)
	}
	err := store.Users().Create(ctx, u2)
	if !errors.Is(err, domain.ErrDuplicateUserEmail) {
		t.Fatalf("want ErrDuplicateUserEmail, got %v", err)
	}
}

func TestPG_Account_CreateFindList(t *testing.T) {
	store, ctx := setupStore(t)
	u, _ := domain.NewUser("A", "a@example.com")
	_ = store.Users().Create(ctx, u)
	a := domain.NewAccount(u.ID, domain.FlavorVanilla)
	if err := store.Accounts().Create(ctx, a); err != nil {
		t.Fatal(err)
	}
	if a.ID == "" {
		t.Fatal("id empty")
	}
	// 二重開設はエラー
	a2 := domain.NewAccount(u.ID, domain.FlavorVanilla)
	if err := store.Accounts().Create(ctx, a2); !errors.Is(err, domain.ErrDuplicateAccount) {
		t.Fatalf("want ErrDuplicateAccount, got %v", err)
	}
	list, err := store.Accounts().ListByUser(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 account, got %d", len(list))
	}
}

func TestPG_UpdateBalance_Insufficient(t *testing.T) {
	store, ctx := setupStore(t)
	u, _ := domain.NewUser("A", "a@example.com")
	_ = store.Users().Create(ctx, u)
	a := domain.NewAccount(u.ID, domain.FlavorVanilla)
	_ = store.Accounts().Create(ctx, a)
	// 残高 0 から -1 しようとすると CHECK 違反
	err := store.Accounts().UpdateBalance(ctx, a.ID, -1)
	if !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatalf("want ErrInsufficientBalance, got %v", err)
	}
}

func TestPG_WithTx_RollbackOnError(t *testing.T) {
	store, ctx := setupStore(t)
	u, _ := domain.NewUser("A", "a@example.com")
	_ = store.Users().Create(ctx, u)
	a := domain.NewAccount(u.ID, domain.FlavorVanilla)
	_ = store.Accounts().Create(ctx, a)
	// 入金して 10 マアム作り、Tx 中に残高を崩す処理をして最後にエラー → ロールバックされる
	_ = store.Accounts().UpdateBalance(ctx, a.ID, 10)

	sentinel := errors.New("boom")
	_ = store.WithTx(ctx, func(ctx context.Context, tx Store) error {
		if err := tx.Accounts().UpdateBalance(ctx, a.ID, -5); err != nil {
			return err
		}
		return sentinel
	})
	got, _ := store.Accounts().FindByID(ctx, a.ID)
	if got.Balance != 10 {
		t.Fatalf("rollback failed: balance=%d, want 10", got.Balance)
	}
}
