package auth_test

import (
	"context"
	"github.com/11DingKing/lab-scheduling/internal/auth"
	"github.com/11DingKing/lab-scheduling/internal/clock"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"github.com/11DingKing/lab-scheduling/internal/testutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPasswordHash(t *testing.T) {
	h := auth.HashPassword("secret")
	if !auth.CheckPassword(h, "secret") {
		t.Fatal("password rejected")
	}
	if auth.CheckPassword(h, "wrong") {
		t.Fatal("wrong password accepted")
	}
}
func TestLoginAuthenticateAndLogout(t *testing.T) {
	db := testutil.Open(t)
	svc := auth.Service{Store: db, Clock: clock.Real{}, TTL: time.Hour}
	token, user, err := svc.Login(context.Background(), "admin@example.com", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || user.Role != domain.RoleAdmin {
		t.Fatalf("token=%q user=%+v", token, user)
	}
	ctx, got, err := svc.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != user.ID {
		t.Fatal(got)
	}
	if _, ok := auth.UserFromContext(ctx); !ok {
		t.Fatal("user absent")
	}
	if err := svc.Logout(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Authenticate(context.Background(), token); err == nil {
		t.Fatal("revoked token accepted")
	}
}
func TestLoginNormalizesEmail(t *testing.T) {
	db := testutil.Open(t)
	svc := auth.Service{Store: db, Clock: clock.Real{}, TTL: time.Hour}
	_, user, err := svc.Login(context.Background(), " ADMIN@example.com ", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "admin@example.com" {
		t.Fatal(user.Email)
	}
}
func TestExpiry(t *testing.T) {
	db := testutil.Open(t)
	now := time.Now().UTC()
	svc := auth.Service{Store: db, Clock: clock.Fixed{Value: now}, TTL: -time.Minute}
	token, _, err := svc.Login(context.Background(), "admin@example.com", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = svc.Authenticate(context.Background(), token); err != domain.ErrExpired {
		t.Fatalf("err=%v", err)
	}
}
func TestTokenFromRequest(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer abc")
	if got := auth.TokenFromRequest(r); got != "abc" {
		t.Fatal(got)
	}
	r = httptest.NewRequest("GET", "/", nil)
	c := (&httpCookie{Name: "lab_session", Value: "cookie"}).HTTP()
	r.AddCookie(c)
	if got := auth.TokenFromRequest(r); got != "cookie" {
		t.Fatal(got)
	}
}

type httpCookie struct{ Name, Value string }

func (c *httpCookie) HTTP() *http.Cookie { return &http.Cookie{Name: c.Name, Value: c.Value} }
