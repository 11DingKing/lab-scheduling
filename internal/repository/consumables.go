package repository

import (
	"context"
	"database/sql"
	"github.com/11DingKing/lab-scheduling/internal/domain"
)

func (d *DB) CreateConsumable(ctx context.Context, c domain.Consumable) error {
	_, err := d.SQL.ExecContext(ctx, "INSERT INTO consumables(id,name,unit,quota,available,version) VALUES(?,?,?,?,?,?)", c.ID, c.Name, c.Unit, c.Quota, c.Available, c.Version)
	return err
}
func (d *DB) ConsumeTx(ctx context.Context, tx *sql.Tx, reservationID, consumableID string, quantity int) error {
	var available, version int
	if err := tx.QueryRowContext(ctx, "SELECT available,version FROM consumables WHERE id=?", consumableID).Scan(&available, &version); err != nil {
		return err
	}
	if available < quantity {
		return domain.ErrCapacity
	}
	res, err := tx.ExecContext(ctx, "UPDATE consumables SET available=available-?,version=version+1 WHERE id=? AND version=?", quantity, consumableID, version)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrConflict
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO reservation_consumables(reservation_id,consumable_id,quantity) VALUES(?,?,?)", reservationID, consumableID, quantity)
	return err
}
func (d *DB) RestoreConsumablesTx(ctx context.Context, tx *sql.Tx, reservationID string) error {
	rows, err := tx.QueryContext(ctx, "SELECT consumable_id,quantity FROM reservation_consumables WHERE reservation_id=?", reservationID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type item struct {
		id string
		q  int
	}
	items := []item{}
	for rows.Next() {
		var i item
		if err := rows.Scan(&i.id, &i.q); err != nil {
			return err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, i := range items {
		if _, err := tx.ExecContext(ctx, "UPDATE consumables SET available=available+?,version=version+1 WHERE id=?", i.q, i.id); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM reservation_consumables WHERE reservation_id=?", reservationID)
	return err
}
