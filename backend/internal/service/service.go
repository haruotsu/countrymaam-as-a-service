package service

import (
	"context"
	"fmt"

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/domain"
	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/repository"
)

// Service はユースケース層。HTTP ハンドラと DB の間を繋ぐ。
// ほとんどのメソッドは viewer（呼び出しているユーザー）の ID を受け取り、
// 他ユーザーの資産を勝手に操作できないように所有権をチェックする。
type Service struct {
	store repository.Store
}

func New(store repository.Store) *Service { return &Service{store: store} }

// ---------- Users（参照のみ） ----------

func (s *Service) GetUser(ctx context.Context, id string) (*domain.User, error) {
	return s.store.Users().FindByID(ctx, id)
}

// FindAccountByEmailAndFlavor は送金先解決のための公開検索。
// email と flavor から相手の口座 ID を得る。見つからなければ ErrAccountNotFound。
func (s *Service) FindAccountByEmailAndFlavor(ctx context.Context, email string, flavor domain.Flavor) (*domain.Account, *domain.User, error) {
	u, err := s.store.Users().FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		return nil, nil, domain.ErrAccountNotFound
	}
	a, err := s.store.Accounts().FindByUserIDAndFlavor(ctx, u.ID, flavor)
	if err != nil {
		return nil, nil, err
	}
	return a, u, nil
}

// ---------- Accounts ----------

// OpenAccount は viewer 本人の口座を開設する。
func (s *Service) OpenAccount(ctx context.Context, viewerID string, flavor domain.Flavor) (*domain.Account, error) {
	a := domain.NewAccount(viewerID, flavor)
	if err := s.store.Accounts().Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) ListMyAccounts(ctx context.Context, viewerID string) ([]*domain.Account, error) {
	return s.store.Accounts().ListByUser(ctx, viewerID)
}

// GetMyAccount は viewer 本人の口座のみ返す。他人のものを指定すると ErrForbidden。
func (s *Service) GetMyAccount(ctx context.Context, viewerID, accountID string) (*domain.Account, error) {
	a, err := s.store.Accounts().FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if a.UserID != viewerID {
		return nil, domain.ErrForbidden
	}
	return a, nil
}

// ---------- Deposit / Withdraw ----------

func (s *Service) Deposit(ctx context.Context, viewerID, accountID string, amount int64, memo string) (*domain.Account, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	var updated *domain.Account
	err := s.store.WithTx(ctx, func(ctx context.Context, tx repository.Store) error {
		if err := ensureOwner(ctx, tx, viewerID, accountID); err != nil {
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

func (s *Service) Withdraw(ctx context.Context, viewerID, accountID string, amount int64, memo string) (*domain.Account, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	var updated *domain.Account
	err := s.store.WithTx(ctx, func(ctx context.Context, tx repository.Store) error {
		if err := ensureOwner(ctx, tx, viewerID, accountID); err != nil {
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
	ViewerID      string
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
		if from.UserID != in.ViewerID {
			return domain.ErrForbidden
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
		return tx.Transactions().Create(ctx, &domain.Transaction{
			AccountID:             to.ID,
			CounterpartyAccountID: &from.ID,
			Type:                  domain.TxTransferIn,
			Amount:                in.Amount,
			Memo:                  in.Memo,
		})
	})
}

// ---------- Exchange ----------

type ExchangeInput struct {
	ViewerID      string
	FromAccountID string
	ToAccountID   string
	Amount        int64
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
		if from.UserID != in.ViewerID {
			return domain.ErrForbidden
		}
		to, err := tx.Accounts().FindByID(ctx, in.ToAccountID)
		if err != nil {
			return err
		}
		if to.UserID != in.ViewerID {
			return domain.ErrForbidden
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

func (s *Service) ListMyTransactions(ctx context.Context, viewerID, accountID string, limit int) ([]*domain.Transaction, error) {
	if _, err := s.GetMyAccount(ctx, viewerID, accountID); err != nil {
		return nil, err
	}
	return s.store.Transactions().ListByAccount(ctx, accountID, limit)
}

// ---------- helpers ----------

func ensureOwner(ctx context.Context, store repository.Store, viewerID, accountID string) error {
	a, err := store.Accounts().FindByID(ctx, accountID)
	if err != nil {
		return err
	}
	if a.UserID != viewerID {
		return domain.ErrForbidden
	}
	return nil
}
