package migration_test

import (
	"context"
	"database/sql"
	"github.com/11DingKing/lab-scheduling/internal/migration"
	"github.com/11DingKing/lab-scheduling/internal/repository"
	"github.com/11DingKing/lab-scheduling/internal/testutil"
	"path/filepath"
	"testing"
)

func TestApplyOnFreshDatabase(t *testing.T) {
	db := testutil.Open(t)
	var n int
	if err := db.SQL.QueryRow("SELECT count(*) FROM schema_migrations").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal(n)
	}
}
func TestApplyMissingDirectory(t *testing.T) {
	db, err := repository.Open(filepath.Join(t.TempDir(), "missing.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = migration.Apply(context.Background(), db.SQL, filepath.Join(t.TempDir(), "none")); err == nil {
		t.Fatal("missing migration accepted")
	}
}
func TestMigrationRollbackOnInvalidSQL(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	tx, err := db.SQL.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, "CREATE TABLE invalid_temp (id TEXT)"); err != nil {
		t.Fatal(err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var n int
	err = db.SQL.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='invalid_temp'").Scan(&n)
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("rollback left table")
	}
}
