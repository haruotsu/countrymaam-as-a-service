package service

import (
	"context"
	"time"

	"github.com/haruotsu/countrymaam-as-a-service/backend/internal/domain"
)

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

// Register はユーザー作成 + ログイン（セッション発行）まで一気に行う。
func (s *Service) Register(ctx context.Context, in RegisterInput) (*domain.User, *domain.Session, error) {
	u, err := domain.NewUser(in.Name, in.Email)
	if err != nil {
		return nil, nil, err
	}
	hash, err := domain.HashPassword(in.Password)
	if err != nil {
		return nil, nil, err
	}
	u.PasswordHash = hash
	if err := s.store.Users().Create(ctx, u); err != nil {
		return nil, nil, err
	}
	sess, err := s.issueSession(ctx, u.ID)
	if err != nil {
		return nil, nil, err
	}
	return u, sess, nil
}

type LoginInput struct {
	Email    string
	Password string
}

// Login は認証成功時に新しいセッションを発行する。
// 誤ったメール／パスワードの区別はクライアントに漏らさず、どちらも ErrInvalidCredentials を返す。
func (s *Service) Login(ctx context.Context, in LoginInput) (*domain.User, *domain.Session, error) {
	u, err := s.store.Users().FindByEmail(ctx, normalizeEmail(in.Email))
	if err != nil {
		// ユーザー不在でも認証失敗として扱う
		return nil, nil, domain.ErrInvalidCredentials
	}
	if err := domain.VerifyPassword(u.PasswordHash, in.Password); err != nil {
		return nil, nil, domain.ErrInvalidCredentials
	}
	sess, err := s.issueSession(ctx, u.ID)
	if err != nil {
		return nil, nil, err
	}
	return u, sess, nil
}

// Logout はトークンに対応するセッションを削除する。存在しなくてもエラーにしない（冪等）。
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.store.Sessions().DeleteByToken(ctx, token)
}

// Authenticate はトークンからユーザーを引く。ミドルウェアから呼ばれる。
func (s *Service) Authenticate(ctx context.Context, token string) (*domain.User, error) {
	sess, err := s.store.Sessions().FindActive(ctx, token)
	if err != nil {
		return nil, domain.ErrUnauthenticated
	}
	u, err := s.store.Users().FindByID(ctx, sess.UserID)
	if err != nil {
		return nil, domain.ErrUnauthenticated
	}
	return u, nil
}

func (s *Service) issueSession(ctx context.Context, userID string) (*domain.Session, error) {
	tok, err := domain.NewSessionToken()
	if err != nil {
		return nil, err
	}
	sess := &domain.Session{
		UserID:    userID,
		Token:     tok,
		ExpiresAt: time.Now().Add(domain.SessionTTL),
	}
	if err := s.store.Sessions().Create(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func normalizeEmail(e string) string {
	return lower(trim(e))
}

// 小さなヘルパ（標準ライブラリに依存させすぎないため）
func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}
func lower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
