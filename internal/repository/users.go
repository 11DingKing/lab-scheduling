package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"time"
)

func (d *DB) FindUserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	var u domain.User
	var hash string
	var active int
	err := d.SQL.QueryRowContext(ctx, "SELECT id,email,password_hash,role,display_name,active FROM users WHERE email=?", email).Scan(&u.ID, &u.Email, &hash, &u.Role, &u.DisplayName, &active)
	if err == sql.ErrNoRows {
		return u, "", domain.ErrNotFound
	}
	if err != nil {
		return u, "", fmt.Errorf("find user: %w", err)
	}
	u.Active = active == 1
	return u, hash, nil
}
func (d *DB) CreateSession(ctx context.Context, userID, token string, created, expires time.Time) (string, error) {
	id := fmt.Sprintf("s-%d", created.UnixNano())
	_, err := d.SQL.ExecContext(ctx, "INSERT INTO sessions(id,user_id,token_hash,expires_at,created_at) VALUES(?,?,?,?,?)", id, userID, token, expires.Format(time.RFC3339Nano), created.Format(time.RFC3339Nano))
	return id, err
}
func (d *DB) FindSession(ctx context.Context, token string) (domain.User, time.Time, error) {
	var u domain.User
	var expires string
	var active int
	err := d.SQL.QueryRowContext(ctx, "SELECT u.id,u.email,u.role,u.display_name,u.active,s.expires_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=? AND s.revoked_at IS NULL", token).Scan(&u.ID, &u.Email, &u.Role, &u.DisplayName, &active, &expires)
	if err == sql.ErrNoRows {
		return u, time.Time{}, domain.ErrForbidden
	}
	if err != nil {
		return u, time.Time{}, err
	}
	u.Active = active == 1
	t, err := time.Parse(time.RFC3339Nano, expires)
	return u, t, err
}
func (d *DB) RevokeSession(ctx context.Context, token string) error {
	_, err := d.SQL.ExecContext(ctx, "UPDATE sessions SET revoked_at=datetime('now') WHERE token_hash=? AND revoked_at IS NULL", token)
	return err
}
