package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/11DingKing/lab-scheduling/internal/domain"
)

func (d *DB) CreateEquipment(ctx context.Context, e domain.Equipment) error {
	_, err := d.SQL.ExecContext(ctx, "INSERT INTO equipment(id,name,kind,capacity,status,version,created_at,updated_at) VALUES(?,?,?,?,?,?,datetime('now'),datetime('now'))", e.ID, e.Name, e.Kind, e.Capacity, e.Status, e.Version)
	return err
}
func (d *DB) Equipment(ctx context.Context, id string) (domain.Equipment, error) {
	var e domain.Equipment
	err := d.SQL.QueryRowContext(ctx, "SELECT id,name,kind,capacity,status,version FROM equipment WHERE id=?", id).Scan(&e.ID, &e.Name, &e.Kind, &e.Capacity, &e.Status, &e.Version)
	if err == sql.ErrNoRows {
		return e, domain.ErrNotFound
	}
	return e, err
}
func (d *DB) UpdateEquipmentStatus(ctx context.Context, id string, status domain.EquipmentStatus, version int) error {
	res, err := d.SQL.ExecContext(ctx, "UPDATE equipment SET status=?,version=version+1,updated_at=datetime('now') WHERE id=? AND version=?", status, id, version)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrConflict
	}
	return nil
}
func (d *DB) ListEquipment(ctx context.Context, status string) ([]domain.Equipment, error) {
	q := "SELECT id,name,kind,capacity,status,version FROM equipment"
	args := []any{}
	if status != "" {
		q += " WHERE status=?"
		args = append(args, status)
	}
	q += " ORDER BY name"
	rows, err := d.SQL.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Equipment, 0)
	for rows.Next() {
		var e domain.Equipment
		if err := rows.Scan(&e.ID, &e.Name, &e.Kind, &e.Capacity, &e.Status, &e.Version); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func overlapsSQL() string { return "starts_at < ? AND ends_at > ?" }
func ensureEquipment(e domain.Equipment) error {
	if e.ID == "" || e.Capacity <= 0 {
		return fmt.Errorf("invalid equipment")
	}
	return nil
}
