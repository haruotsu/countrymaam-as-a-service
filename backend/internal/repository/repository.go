package repository

import (
	"context"

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/domain"
)

// Store はアプリが必要とする永続化の全ての入口。
// 単一リクエストの範囲で使う（プール経由）。
type Store interface {
	Users() UserRepo
	Accounts() AccountRepo
	Transactions() TransactionRepo
	Sessions() SessionRepo

	// WithTx は fn をトランザクション内で実行する。fn がエラーを返せばロールバック。
	// fn に渡される Store はそのトランザクションにバインドされている。
	WithTx(ctx context.Context, fn func(ctx context.Context, tx Store) error) error
}

type UserRepo interface {
	Create(ctx context.Context, u *domain.User) error
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context) ([]*domain.User, error)
}

type AccountRepo interface {
	Create(ctx context.Context, a *domain.Account) error
	FindByID(ctx context.Context, id string) (*domain.Account, error)
	FindByUserIDAndFlavor(ctx context.Context, userID string, flavor domain.Flavor) (*domain.Account, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.Account, error)

	// UpdateBalance は id の残高を delta だけ加算する（delta は負でも可）。
	// 実行後の残高が負になる場合 ErrInsufficientBalance を返す。
	UpdateBalance(ctx context.Context, id string, delta int64) error
}

type TransactionRepo interface {
	Create(ctx context.Context, t *domain.Transaction) error
	ListByAccount(ctx context.Context, accountID string, limit int) ([]*domain.Transaction, error)
}

type SessionRepo interface {
	Create(ctx context.Context, s *domain.Session) error
	// FindActive はトークンで有効期限内のセッションを探す。期限切れは ErrSessionNotFound。
	FindActive(ctx context.Context, token string) (*domain.Session, error)
	DeleteByToken(ctx context.Context, token string) error
}
