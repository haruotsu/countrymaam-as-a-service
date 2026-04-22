package domain

import "time"

// Account はユーザーが持つ「フレーバー別の金庫」を表す。残高はマアム枚数（整数）。
type Account struct {
	ID        string
	UserID    string
	Flavor    Flavor
	Balance   int64
	CreatedAt time.Time
}

// NewAccount は残高 0 の新規口座を生成する（ID/CreatedAt は未設定。永続化時に付与）。
func NewAccount(userID string, flavor Flavor) *Account {
	return &Account{
		UserID:  userID,
		Flavor:  flavor,
		Balance: 0,
	}
}

// Deposit は amount マアムを預け入れる。
func (a *Account) Deposit(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	a.Balance += amount
	return nil
}

// Withdraw は amount マアムを引き出す。残高不足ならエラーで残高は変化しない。
func (a *Account) Withdraw(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.Balance < amount {
		return ErrInsufficientBalance
	}
	a.Balance -= amount
	return nil
}
