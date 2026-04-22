package domain

import "errors"

var (
	ErrInvalidAmount        = errors.New("amount must be positive")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrExchangeTooSmall     = errors.New("amount too small to exchange")
	ErrFlavorMismatch       = errors.New("flavor mismatch between accounts")
	ErrSelfTransfer         = errors.New("cannot transfer to the same account")
	ErrForeignExchange      = errors.New("exchange must be within the same user's accounts")
	ErrAccountNotFound      = errors.New("account not found")
	ErrUserNotFound         = errors.New("user not found")
	ErrDuplicateUserEmail   = errors.New("duplicate user email")
	ErrDuplicateAccount     = errors.New("account already exists for this user and flavor")
)
