package domain

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

// SessionTTL はログイン後のセッションの有効期間。パロディ銀行なので長めに。
const SessionTTL = 7 * 24 * time.Hour

type Session struct {
	ID        string
	UserID    string
	Token     string // DB にはこのトークンをそのまま保存（ランダム 32byte）
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (s *Session) IsExpired(now time.Time) bool {
	return !s.ExpiresAt.After(now)
}

// NewSessionToken は 32byte の crypto/rand で生成した token を base64url で返す。
// 長さは 43 文字程度。短いトークンはセッション固定化のリスクを上げるので使わない。
func NewSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
