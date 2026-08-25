package migration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func Apply(ctx context.Context, db *sql.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	if _, err = db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)"); err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, name := range files {
		parts := strings.Split(name, "_")
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		var count int
		if err = db.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations WHERE version=?", v).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version,applied_at) VALUES(?,datetime('now'))", v); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return seed(ctx, db)
}
func seed(ctx context.Context, db *sql.DB) error {
	var n int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	users := []struct{ id, email, role, name string }{{"u-teacher", "teacher@example.com", "teacher", "Teacher One"}, {"u-student", "student@example.com", "student", "Student One"}, {"u-admin", "admin@example.com", "admin", "Lab Admin"}, {"u-safety", "safety@example.com", "safety", "Safety Lead"}}
	for _, u := range users {
		if _, err := db.ExecContext(ctx, "INSERT INTO users(id,email,password_hash,role,display_name,created_at) VALUES(?,?,?,?,?,datetime('now'))", u.id, u.email, hash(u.role), u.role, u.name); err != nil {
			return err
		}
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO equipment(id,name,kind,capacity,status,version,created_at,updated_at) VALUES('eq-cnc','CNC Mill A','cnc',12,'ready',1,datetime('now'),datetime('now'))"); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, "INSERT INTO consumables(id,name,unit,quota,available,version) VALUES('mat-al','Aluminium blanks','piece',500,500,1)")
	return err
}
func hash(role string) string {
	sum := sha256.Sum256([]byte("lab-scheduling:" + role))
	return fmt.Sprintf("%x", sum[:])
}
