package repository

import (
	"context"
	"database/sql"
)

func (d *DB) RecoverExpiredSessions(ctx context.Context) (int64, error) {
	res, err := d.SQL.ExecContext(ctx, "UPDATE sessions SET revoked_at=datetime('now') WHERE revoked_at IS NULL AND expires_at<=datetime('now')")
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
func (d *DB) ActiveReservationCount(ctx context.Context, equipmentID string) (int, error) {
	var n int
	err := d.SQL.QueryRowContext(ctx, "SELECT count(*) FROM reservations WHERE equipment_id=? AND status IN ('held','checked_out','suspended')", equipmentID).Scan(&n)
	return n, err
}
func (d *DB) PingContext(ctx context.Context) error { return d.SQL.PingContext(ctx) }
func rollback(tx *sql.Tx, err error) error {
	if err != nil {
		_ = tx.Rollback()
	}
	return err
}
