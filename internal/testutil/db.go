package testutil

import (
	"context"
	"github.com/11DingKing/lab-scheduling/internal/migration"
	"github.com/11DingKing/lab-scheduling/internal/repository"
	"path/filepath"
	"runtime"
	"testing"
)

func Open(t *testing.T) *repository.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := repository.Open(filepath.Join(dir, "lab.db"))
	if err != nil {
		t.Fatal(err)
	}
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	if err = migration.Apply(context.Background(), db.SQL, filepath.Join(root, "migrations")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
