package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/domain"
)

// NewMemoryStore はテスト用のインメモリ Store。
// WithTx はスナップショット + ロールバック方式で原子性を再現する。
func NewMemoryStore() Store {
	return &memStore{
		data: &memData{
			users:        map[string]*domain.User{},
			accounts:     map[string]*domain.Account{},
			transactions: []*domain.Transaction{},
		},
		nextID: &atomicCounter{},
	}
}

type memData struct {
	mu           sync.Mutex
	users        map[string]*domain.User
	accounts     map[string]*domain.Account
	transactions []*domain.Transaction
}

type atomicCounter struct {
	mu sync.Mutex
	n  int
}

func (c *atomicCounter) next(prefix string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	return fmt.Sprintf("%s-%d", prefix, c.n)
}

type memStore struct {
	data   *memData
	nextID *atomicCounter
	// inTx が true のときは WithTx を再帰させない
	inTx bool
}

func (s *memStore) Users() UserRepo               { return &memUserRepo{s: s} }
func (s *memStore) Accounts() AccountRepo         { return &memAccountRepo{s: s} }
func (s *memStore) Transactions() TransactionRepo { return &memTxRepo{s: s} }

func (s *memStore) WithTx(ctx context.Context, fn func(ctx context.Context, tx Store) error) error {
	if s.inTx {
		return fmt.Errorf("nested transaction is not supported")
	}
	s.data.mu.Lock()
	defer s.data.mu.Unlock()

	// スナップショット
	snapshot := &memData{
		users:        map[string]*domain.User{},
		accounts:     map[string]*domain.Account{},
		transactions: make([]*domain.Transaction, 0, len(s.data.transactions)),
	}
	for k, v := range s.data.users {
		cp := *v
		snapshot.users[k] = &cp
	}
	for k, v := range s.data.accounts {
		cp := *v
		snapshot.accounts[k] = &cp
	}
	for _, t := range s.data.transactions {
		cp := *t
		snapshot.transactions = append(snapshot.transactions, &cp)
	}

	// tx スコープでは data を差し替え、ロック不要にする
	scoped := &memStore{data: s.data, nextID: s.nextID, inTx: true}
	if err := fn(ctx, &noLockStore{inner: scoped}); err != nil {
		// ロールバック
		s.data.users = snapshot.users
		s.data.accounts = snapshot.accounts
		s.data.transactions = snapshot.transactions
		return err
	}
	return nil
}

// noLockStore はロック済みの状態で再ロックを避けるためのラッパ。
type noLockStore struct{ inner *memStore }

func (n *noLockStore) Users() UserRepo               { return &memUserRepo{s: n.inner, locked: true} }
func (n *noLockStore) Accounts() AccountRepo         { return &memAccountRepo{s: n.inner, locked: true} }
func (n *noLockStore) Transactions() TransactionRepo { return &memTxRepo{s: n.inner, locked: true} }
func (n *noLockStore) WithTx(ctx context.Context, fn func(ctx context.Context, tx Store) error) error {
	return fmt.Errorf("nested transaction is not supported")
}

// --- users ---

type memUserRepo struct {
	s      *memStore
	locked bool
}

func (r *memUserRepo) lock()   { if !r.locked { r.s.data.mu.Lock() } }
func (r *memUserRepo) unlock() { if !r.locked { r.s.data.mu.Unlock() } }

func (r *memUserRepo) Create(ctx context.Context, u *domain.User) error {
	r.lock()
	defer r.unlock()
	for _, v := range r.s.data.users {
		if v.Email == u.Email {
			return domain.ErrDuplicateUserEmail
		}
	}
	u.ID = r.s.nextID.next("u")
	u.CreatedAt = time.Now()
	cp := *u
	r.s.data.users[u.ID] = &cp
	return nil
}

func (r *memUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	r.lock()
	defer r.unlock()
	u, ok := r.s.data.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *memUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.lock()
	defer r.unlock()
	for _, v := range r.s.data.users {
		if v.Email == email {
			cp := *v
			return &cp, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *memUserRepo) List(ctx context.Context) ([]*domain.User, error) {
	r.lock()
	defer r.unlock()
	out := make([]*domain.User, 0, len(r.s.data.users))
	for _, v := range r.s.data.users {
		cp := *v
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

// --- accounts ---

type memAccountRepo struct {
	s      *memStore
	locked bool
}

func (r *memAccountRepo) lock()   { if !r.locked { r.s.data.mu.Lock() } }
func (r *memAccountRepo) unlock() { if !r.locked { r.s.data.mu.Unlock() } }

func (r *memAccountRepo) Create(ctx context.Context, a *domain.Account) error {
	r.lock()
	defer r.unlock()
	if _, ok := r.s.data.users[a.UserID]; !ok {
		return domain.ErrUserNotFound
	}
	for _, v := range r.s.data.accounts {
		if v.UserID == a.UserID && v.Flavor == a.Flavor {
			return domain.ErrDuplicateAccount
		}
	}
	a.ID = r.s.nextID.next("a")
	a.CreatedAt = time.Now()
	cp := *a
	r.s.data.accounts[a.ID] = &cp
	return nil
}

func (r *memAccountRepo) FindByID(ctx context.Context, id string) (*domain.Account, error) {
	r.lock()
	defer r.unlock()
	a, ok := r.s.data.accounts[id]
	if !ok {
		return nil, domain.ErrAccountNotFound
	}
	cp := *a
	return &cp, nil
}

func (r *memAccountRepo) FindByUserIDAndFlavor(ctx context.Context, userID string, flavor domain.Flavor) (*domain.Account, error) {
	r.lock()
	defer r.unlock()
	for _, v := range r.s.data.accounts {
		if v.UserID == userID && v.Flavor == flavor {
			cp := *v
			return &cp, nil
		}
	}
	return nil, domain.ErrAccountNotFound
}

func (r *memAccountRepo) ListByUser(ctx context.Context, userID string) ([]*domain.Account, error) {
	r.lock()
	defer r.unlock()
	var out []*domain.Account
	for _, v := range r.s.data.accounts {
		if v.UserID == userID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *memAccountRepo) UpdateBalance(ctx context.Context, id string, delta int64) error {
	r.lock()
	defer r.unlock()
	a, ok := r.s.data.accounts[id]
	if !ok {
		return domain.ErrAccountNotFound
	}
	if a.Balance+delta < 0 {
		return domain.ErrInsufficientBalance
	}
	a.Balance += delta
	return nil
}

// --- transactions ---

type memTxRepo struct {
	s      *memStore
	locked bool
}

func (r *memTxRepo) lock()   { if !r.locked { r.s.data.mu.Lock() } }
func (r *memTxRepo) unlock() { if !r.locked { r.s.data.mu.Unlock() } }

func (r *memTxRepo) Create(ctx context.Context, t *domain.Transaction) error {
	r.lock()
	defer r.unlock()
	if _, ok := r.s.data.accounts[t.AccountID]; !ok {
		return domain.ErrAccountNotFound
	}
	t.ID = r.s.nextID.next("t")
	t.CreatedAt = time.Now()
	cp := *t
	r.s.data.transactions = append(r.s.data.transactions, &cp)
	return nil
}

func (r *memTxRepo) ListByAccount(ctx context.Context, accountID string, limit int) ([]*domain.Transaction, error) {
	r.lock()
	defer r.unlock()
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var out []*domain.Transaction
	for _, t := range r.s.data.transactions {
		if t.AccountID == accountID {
			cp := *t
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
