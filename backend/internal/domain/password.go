package domain

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// minPasswordLen はうっかり "pass" 等を登録させないためだけの下限。
// 長さで強さは担保できないが、明らかに短いものを弾くためのガード。
const minPasswordLen = 8

// HashPassword は平文を bcrypt ハッシュに変換する。
func HashPassword(plain string) (string, error) {
	if len(plain) < minPasswordLen {
		return "", ErrWeakPassword
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword はハッシュと平文が一致するか確認する。
// 不一致は ErrInvalidCredentials として返し、ユーザー列挙攻撃を避けやすくする。
func VerifyPassword(hash, plain string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrInvalidCredentials
	}
	return err
}
