package domain

import (
	"errors"
	"strings"
	"time"
)

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

var ErrInvalidUser = errors.New("invalid user")

func NewUser(name, email string) (*User, error) {
	n := strings.TrimSpace(name)
	e := strings.TrimSpace(strings.ToLower(email))
	if n == "" || e == "" || !strings.Contains(e, "@") {
		return nil, ErrInvalidUser
	}
	return &User{Name: n, Email: e}, nil
}

// TxType は取引種別。
type TxType string

const (
	TxDeposit     TxType = "deposit"
	TxWithdraw    TxType = "withdraw"
	TxTransferIn  TxType = "transfer_in"
	TxTransferOut TxType = "transfer_out"
	TxExchangeIn  TxType = "exchange_in"
	TxExchangeOut TxType = "exchange_out"
)

// Transaction は口座単位に記録される取引レコード。
type Transaction struct {
	ID                    string
	AccountID             string
	CounterpartyAccountID *string
	Type                  TxType
	Amount                int64
	Memo                  string
	CreatedAt             time.Time
}
