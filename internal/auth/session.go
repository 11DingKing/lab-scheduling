package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/11DingKing/lab-scheduling/internal/clock"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"net/http"
	"strings"
	"time"
)

type Store interface {
	CreateSession(context.Context, string, string, time.Time, time.Time) (string, error)
	FindSession(context.Context, string) (domain.User, time.Time, error)
	RevokeSession(context.Context, string) error
}
type Service struct {
	Store Store
	Clock clock.Clock
	TTL   time.Duration
}
type ctxKey string

const userKey ctxKey = "authenticated-user"

func (s Service) Login(ctx context.Context, email, password string) (string, domain.User, error) {
	user, err := s.StoreUser(ctx, domain.NormalizeEmail(email), password)
	if err != nil {
		return "", domain.User{}, err
	}
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", domain.User{}, fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(raw)
	now := s.Clock.Now()
	id, err := s.Store.CreateSession(ctx, user.ID, hashToken(token), now, now.Add(s.TTL))
	if err != nil {
		return "", domain.User{}, fmt.Errorf("create session: %w", err)
	}
	_ = id
	return token, user, nil
}

type userStore interface {
	FindUserByEmail(context.Context, string) (domain.User, string, error)
}

func (s Service) StoreUser(ctx context.Context, email, password string) (domain.User, error) {
	st, ok := s.Store.(userStore)
	if !ok {
		return domain.User{}, errors.New("auth store unavailable")
	}
	user, hash, err := st.FindUserByEmail(ctx, email)
	if err != nil {
		return domain.User{}, err
	}
	if !user.Active || !CheckPassword(hash, password) {
		return domain.User{}, domain.ErrForbidden
	}
	return user, nil
}
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
func (s Service) Authenticate(ctx context.Context, token string) (context.Context, domain.User, error) {
	if strings.TrimSpace(token) == "" {
		return ctx, domain.User{}, domain.ErrForbidden
	}
	user, expires, err := s.Store.FindSession(ctx, hashToken(token))
	if err != nil {
		return ctx, domain.User{}, err
	}
	if !expires.After(s.Clock.Now()) {
		return ctx, domain.User{}, domain.ErrExpired
	}
	return context.WithValue(ctx, userKey, user), user, nil
}
func UserFromContext(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(userKey).(domain.User)
	return user, ok
}
func (s Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.Store.RevokeSession(ctx, hashToken(token))
}
func TokenFromRequest(r *http.Request) string {
	if v := r.Header.Get("Authorization"); strings.HasPrefix(v, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(v, "Bearer "))
	}
	if c, err := r.Cookie("lab_session"); err == nil {
		return c.Value
	}
	return ""
}
