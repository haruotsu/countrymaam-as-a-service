package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// dbtx は *pgxpool.Pool と pgx.Tx の共通インタフェース。
// Store 実装がトランザクション有無の両方で同じ SQL を使えるようにする。
type dbtx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type pgStore struct {
	pool *pgxpool.Pool // トランザクション開始に必要。tx 内では nil。
	q    dbtx          // 実際のクエリ実行者
}

// NewStoreFromURL はコネクションプールを張り Store を返す。
func NewStoreFromURL(ctx context.Context, url string) (Store, func(), error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, err
	}
	s := &pgStore{pool: pool, q: pool}
	return s, pool.Close, nil
}

func (s *pgStore) Users() UserRepo                { return &userRepo{q: s.q} }
func (s *pgStore) Accounts() AccountRepo          { return &accountRepo{q: s.q} }
func (s *pgStore) Transactions() TransactionRepo  { return &txRepo{q: s.q} }

func (s *pgStore) WithTx(ctx context.Context, fn func(ctx context.Context, tx Store) error) error {
	if s.pool == nil {
		return fmt.Errorf("nested transaction is not supported")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // コミット後の Rollback はno-op
	scoped := &pgStore{pool: nil, q: tx}
	if err := fn(ctx, scoped); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ---------- users ----------

type userRepo struct{ q dbtx }

func (r *userRepo) Create(ctx context.Context, u *domain.User) error {
	row := r.q.QueryRow(ctx,
		`INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, created_at`,
		u.Name, u.Email)
	if err := row.Scan(&u.ID, &u.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicateUserEmail
		}
		return err
	}
	return nil
}

func (r *userRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	u := &domain.User{}
	err := r.q.QueryRow(ctx,
		`SELECT id, name, email, created_at FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u := &domain.User{}
	err := r.q.QueryRow(ctx,
		`SELECT id, name, email, created_at FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *userRepo) List(ctx context.Context) ([]*domain.User, error) {
	rows, err := r.q.Query(ctx,
		`SELECT id, name, email, created_at FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.User
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// ---------- accounts ----------

type accountRepo struct{ q dbtx }

func (r *accountRepo) Create(ctx context.Context, a *domain.Account) error {
	row := r.q.QueryRow(ctx,
		`INSERT INTO accounts (user_id, flavor, balance) VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		a.UserID, string(a.Flavor), a.Balance)
	if err := row.Scan(&a.ID, &a.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicateAccount
		}
		if isForeignKeyViolation(err) {
			return domain.ErrUserNotFound
		}
		return err
	}
	return nil
}

func (r *accountRepo) FindByID(ctx context.Context, id string) (*domain.Account, error) {
	a := &domain.Account{}
	var flavor string
	err := r.q.QueryRow(ctx,
		`SELECT id, user_id, flavor, balance, created_at FROM accounts WHERE id = $1`, id).
		Scan(&a.ID, &a.UserID, &flavor, &a.Balance, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAccountNotFound
	}
	if err != nil {
		return nil, err
	}
	a.Flavor = domain.Flavor(flavor)
	return a, nil
}

func (r *accountRepo) FindByUserIDAndFlavor(ctx context.Context, userID string, flavor domain.Flavor) (*domain.Account, error) {
	a := &domain.Account{}
	var f string
	err := r.q.QueryRow(ctx,
		`SELECT id, user_id, flavor, balance, created_at FROM accounts
		 WHERE user_id = $1 AND flavor = $2`, userID, string(flavor)).
		Scan(&a.ID, &a.UserID, &f, &a.Balance, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAccountNotFound
	}
	if err != nil {
		return nil, err
	}
	a.Flavor = domain.Flavor(f)
	return a, nil
}

func (r *accountRepo) ListByUser(ctx context.Context, userID string) ([]*domain.Account, error) {
	rows, err := r.q.Query(ctx,
		`SELECT id, user_id, flavor, balance, created_at FROM accounts
		 WHERE user_id = $1 ORDER BY created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Account
	for rows.Next() {
		a := &domain.Account{}
		var f string
		if err := rows.Scan(&a.ID, &a.UserID, &f, &a.Balance, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.Flavor = domain.Flavor(f)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *accountRepo) UpdateBalance(ctx context.Context, id string, delta int64) error {
	// CHECK(balance >= 0) に当たったときに InsufficientBalance を返す。
	tag, err := r.q.Exec(ctx,
		`UPDATE accounts SET balance = balance + $1 WHERE id = $2`,
		delta, id)
	if err != nil {
		if isCheckViolation(err) {
			return domain.ErrInsufficientBalance
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAccountNotFound
	}
	return nil
}

// ---------- transactions ----------

type txRepo struct{ q dbtx }

func (r *txRepo) Create(ctx context.Context, t *domain.Transaction) error {
	row := r.q.QueryRow(ctx,
		`INSERT INTO transactions (account_id, counterparty_account_id, type, amount, memo)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		t.AccountID, t.CounterpartyAccountID, string(t.Type), t.Amount, t.Memo)
	return row.Scan(&t.ID, &t.CreatedAt)
}

func (r *txRepo) ListByAccount(ctx context.Context, accountID string, limit int) ([]*domain.Transaction, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.q.Query(ctx,
		`SELECT id, account_id, counterparty_account_id, type, amount, memo, created_at
		 FROM transactions WHERE account_id = $1 ORDER BY created_at DESC LIMIT $2`,
		accountID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Transaction
	for rows.Next() {
		t := &domain.Transaction{}
		var typ string
		if err := rows.Scan(&t.ID, &t.AccountID, &t.CounterpartyAccountID, &typ, &t.Amount, &t.Memo, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.Type = domain.TxType(typ)
		out = append(out, t)
	}
	return out, rows.Err()
}

// ---------- helpers ----------

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

// CleanAll はテスト用に全テーブルを TRUNCATE する。本番では呼ばない前提。
func (s *pgStore) CleanAll(ctx context.Context) error {
	_, err := s.q.Exec(ctx, strings.Join([]string{
		`TRUNCATE transactions, accounts, users RESTART IDENTITY CASCADE`,
	}, ";"))
	return err
}
