package service

import (
	"context"
	"fmt"

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/domain"
	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/repository"
)

// Service はユースケース層。HTTP ハンドラと DB の間を繋ぐ。
type Service struct {
	store repository.Store
}

func New(store repository.Store) *Service { return &Service{store: store} }

// ---------- Users ----------

type CreateUserInput struct {
	Name  string
	Email string
}

func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (*domain.User, error) {
	u, err := domain.NewUser(in.Name, in.Email)
	if err != nil {
		return nil, err
	}
	if err := s.store.Users().Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]*domain.User, error) {
	return s.store.Users().List(ctx)
}

func (s *Service) GetUser(ctx context.Context, id string) (*domain.User, error) {
	return s.store.Users().FindByID(ctx, id)
}

// ---------- Accounts ----------

type OpenAccountInput struct {
	UserID string
	Flavor domain.Flavor
}

func (s *Service) OpenAccount(ctx context.Context, in OpenAccountInput) (*domain.Account, error) {
	if _, err := s.store.Users().FindByID(ctx, in.UserID); err != nil {
		return nil, err
	}
	a := domain.NewAccount(in.UserID, in.Flavor)
	if err := s.store.Accounts().Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	return s.store.Accounts().FindByID(ctx, id)
}

func (s *Service) ListAccountsByUser(ctx context.Context, userID string) ([]*domain.Account, error) {
	return s.store.Accounts().ListByUser(ctx, userID)
}

// ---------- Deposit / Withdraw ----------

func (s *Service) Deposit(ctx context.Context, accountID string, amount int64, memo string) (*domain.Account, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	var updated *domain.Account
	err := s.store.WithTx(ctx, func(ctx context.Context, tx repository.Store) error {
		if _, err := tx.Accounts().FindByID(ctx, accountID); err != nil {
			return err
		}
		if err := tx.Accounts().UpdateBalance(ctx, accountID, amount); err != nil {
			return err
		}
		if err := tx.Transactions().Create(ctx, &domain.Transaction{
			AccountID: accountID,
			Type:      domain.TxDeposit,
			Amount:    amount,
			Memo:      memo,
		}); err != nil {
			return err
		}
		u, err := tx.Accounts().FindByID(ctx, accountID)
		if err != nil {
			return err
		}
		updated = u
		return nil
	})
	return updated, err
}

func (s *Service) Withdraw(ctx context.Context, accountID string, amount int64, memo string) (*domain.Account, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	var updated *domain.Account
	err := s.store.WithTx(ctx, func(ctx context.Context, tx repository.Store) error {
		if _, err := tx.Accounts().FindByID(ctx, accountID); err != nil {
			return err
		}
		if err := tx.Accounts().UpdateBalance(ctx, accountID, -amount); err != nil {
			return err
		}
		if err := tx.Transactions().Create(ctx, &domain.Transaction{
			AccountID: accountID,
			Type:      domain.TxWithdraw,
			Amount:    amount,
			Memo:      memo,
		}); err != nil {
			return err
		}
		u, err := tx.Accounts().FindByID(ctx, accountID)
		if err != nil {
			return err
		}
		updated = u
		return nil
	})
	return updated, err
}

// ---------- Transfer ----------

type TransferInput struct {
	FromAccountID string
	ToAccountID   string
	Amount        int64
	Memo          string
}

func (s *Service) Transfer(ctx context.Context, in TransferInput) error {
	if in.Amount <= 0 {
		return domain.ErrInvalidAmount
	}
	if in.FromAccountID == in.ToAccountID {
		return domain.ErrSelfTransfer
	}
	return s.store.WithTx(ctx, func(ctx context.Context, tx repository.Store) error {
		from, err := tx.Accounts().FindByID(ctx, in.FromAccountID)
		if err != nil {
			return fmt.Errorf("from: %w", err)
		}
		to, err := tx.Accounts().FindByID(ctx, in.ToAccountID)
		if err != nil {
			return fmt.Errorf("to: %w", err)
		}
		if from.Flavor != to.Flavor {
			return domain.ErrFlavorMismatch
		}
		if err := tx.Accounts().UpdateBalance(ctx, from.ID, -in.Amount); err != nil {
			return err
		}
		if err := tx.Accounts().UpdateBalance(ctx, to.ID, in.Amount); err != nil {
			return err
		}
		if err := tx.Transactions().Create(ctx, &domain.Transaction{
			AccountID:             from.ID,
			CounterpartyAccountID: &to.ID,
			Type:                  domain.TxTransferOut,
			Amount:                in.Amount,
			Memo:                  in.Memo,
		}); err != nil {
			return err
		}
		if err := tx.Transactions().Create(ctx, &domain.Transaction{
			AccountID:             to.ID,
			CounterpartyAccountID: &from.ID,
			Type:                  domain.TxTransferIn,
			Amount:                in.Amount,
			Memo:                  in.Memo,
		}); err != nil {
			return err
		}
		return nil
	})
}

// ---------- Exchange ----------

type ExchangeInput struct {
	FromAccountID string
	ToAccountID   string
	Amount        int64 // from 側で引く量
	Memo          string
}

type ExchangeResult struct {
	FromAmount int64
	ToAmount   int64
}

func (s *Service) Exchange(ctx context.Context, in ExchangeInput) (*ExchangeResult, error) {
	if in.Amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if in.FromAccountID == in.ToAccountID {
		return nil, domain.ErrSelfTransfer
	}
	var result *ExchangeResult
	err := s.store.WithTx(ctx, func(ctx context.Context, tx repository.Store) error {
		from, err := tx.Accounts().FindByID(ctx, in.FromAccountID)
		if err != nil {
			return err
		}
		to, err := tx.Accounts().FindByID(ctx, in.ToAccountID)
		if err != nil {
			return err
		}
		if from.UserID != to.UserID {
			return domain.ErrForeignExchange
		}
		converted, err := domain.Exchange(from.Flavor, to.Flavor, in.Amount)
		if err != nil {
			return err
		}
		if err := tx.Accounts().UpdateBalance(ctx, from.ID, -in.Amount); err != nil {
			return err
		}
		if err := tx.Accounts().UpdateBalance(ctx, to.ID, converted); err != nil {
			return err
		}
		if err := tx.Transactions().Create(ctx, &domain.Transaction{
			AccountID:             from.ID,
			CounterpartyAccountID: &to.ID,
			Type:                  domain.TxExchangeOut,
			Amount:                in.Amount,
			Memo:                  in.Memo,
		}); err != nil {
			return err
		}
		if err := tx.Transactions().Create(ctx, &domain.Transaction{
			AccountID:             to.ID,
			CounterpartyAccountID: &from.ID,
			Type:                  domain.TxExchangeIn,
			Amount:                converted,
			Memo:                  in.Memo,
		}); err != nil {
			return err
		}
		result = &ExchangeResult{FromAmount: in.Amount, ToAmount: converted}
		return nil
	})
	return result, err
}

// ---------- Transactions ----------

func (s *Service) ListTransactions(ctx context.Context, accountID string, limit int) ([]*domain.Transaction, error) {
	return s.store.Transactions().ListByAccount(ctx, accountID, limit)
}
