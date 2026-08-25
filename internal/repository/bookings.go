package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"time"
)

type BookingRow struct {
	Request   domain.BookingRequest
	Batch     domain.CourseBatch
	Equipment domain.Equipment
}

func (d *DB) CreateBooking(ctx context.Context, r domain.BookingRequest) error {
	_, err := d.SQL.ExecContext(ctx, "INSERT INTO booking_requests(id,batch_id,requester_id,equipment_id,starts_at,ends_at,status,version,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,datetime('now'),datetime('now'))", r.ID, r.BatchID, r.RequesterID, r.EquipmentID, r.Window.Start.Format(time.RFC3339Nano), r.Window.End.Format(time.RFC3339Nano), r.Status, r.Version)
	return err
}
func (d *DB) Booking(ctx context.Context, id string) (domain.BookingRequest, error) {
	var r domain.BookingRequest
	var st, en string
	err := d.SQL.QueryRowContext(ctx, "SELECT id,batch_id,requester_id,equipment_id,starts_at,ends_at,status,version FROM booking_requests WHERE id=?", id).Scan(&r.ID, &r.BatchID, &r.RequesterID, &r.EquipmentID, &st, &en, &r.Status, &r.Version)
	if err == sql.ErrNoRows {
		return r, domain.ErrNotFound
	}
	if err != nil {
		return r, err
	}
	r.Window.Start, _ = time.Parse(time.RFC3339Nano, st)
	r.Window.End, _ = time.Parse(time.RFC3339Nano, en)
	return r, nil
}
func (d *DB) SetBookingStatusTx(ctx context.Context, tx *sql.Tx, id string, from, to domain.RequestStatus, version int) error {
	res, err := tx.ExecContext(ctx, "UPDATE booking_requests SET status=?,version=version+1,updated_at=datetime('now') WHERE id=? AND status=? AND version=?", to, id, from, version)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrConflict
	}
	return nil
}
func (d *DB) HasConflictTx(ctx context.Context, tx *sql.Tx, equipmentID string, w domain.TimeWindow, exclude string) (bool, error) {
	q := "SELECT count(*) FROM reservations WHERE equipment_id=? AND status IN ('held','checked_out','suspended') AND " + overlapsSQL()
	args := []any{equipmentID, w.End.Format(time.RFC3339Nano), w.Start.Format(time.RFC3339Nano)}
	if exclude != "" {
		q += " AND id<>?"
		args = append(args, exclude)
	}
	var n int
	err := tx.QueryRowContext(ctx, q, args...).Scan(&n)
	return n > 0, err
}
func (d *DB) HasMaintenanceTx(ctx context.Context, tx *sql.Tx, equipmentID string, w domain.TimeWindow) (bool, error) {
	var n int
	err := tx.QueryRowContext(ctx, "SELECT count(*) FROM maintenance_blocks WHERE equipment_id=? AND status='active' AND "+overlapsSQL(), equipmentID, w.End.Format(time.RFC3339Nano), w.Start.Format(time.RFC3339Nano)).Scan(&n)
	return n > 0, err
}
func (d *DB) CreateReservationTx(ctx context.Context, tx *sql.Tx, r domain.Reservation) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO reservations(id,request_id,equipment_id,holder_id,starts_at,ends_at,status,version,created_at) VALUES(?,?,?,?,?,?,?,?,datetime('now'))", r.ID, r.RequestID, r.EquipmentID, r.HolderID, r.Window.Start.Format(time.RFC3339Nano), r.Window.End.Format(time.RFC3339Nano), r.Status, r.Version)
	return err
}
func (d *DB) Reservation(ctx context.Context, id string) (domain.Reservation, error) {
	var r domain.Reservation
	var st, en string
	err := d.SQL.QueryRowContext(ctx, "SELECT id,request_id,equipment_id,holder_id,starts_at,ends_at,status,version FROM reservations WHERE id=?", id).Scan(&r.ID, &r.RequestID, &r.EquipmentID, &r.HolderID, &st, &en, &r.Status, &r.Version)
	if err == sql.ErrNoRows {
		return r, domain.ErrNotFound
	}
	if err != nil {
		return r, fmt.Errorf("reservation: %w", err)
	}
	r.Window.Start, _ = time.Parse(time.RFC3339Nano, st)
	r.Window.End, _ = time.Parse(time.RFC3339Nano, en)
	return r, nil
}
func (d *DB) SetReservationStatus(ctx context.Context, id string, from, to domain.ReservationStatus, version int) error {
	res, err := d.SQL.ExecContext(ctx, "UPDATE reservations SET status=?,version=version+1 WHERE id=? AND status=? AND version=?", to, id, from, version)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrConflict
	}
	return nil
}
func (d *DB) SetReservationStatusTx(ctx context.Context, tx *sql.Tx, id string, from, to domain.ReservationStatus, version int) error {
	res, err := tx.ExecContext(ctx, "UPDATE reservations SET status=?,version=version+1 WHERE id=? AND status=? AND version=?", to, id, from, version)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrConflict
	}
	return nil
}
func (d *DB) CreateMaintenance(ctx context.Context, b domain.MaintenanceBlock) error {
	_, err := d.SQL.ExecContext(ctx, "INSERT INTO maintenance_blocks(id,equipment_id,starts_at,ends_at,reason,created_by,status) VALUES(?,?,?,?,?,?,?)", b.ID, b.EquipmentID, b.Window.Start.Format(time.RFC3339Nano), b.Window.End.Format(time.RFC3339Nano), b.Reason, b.CreatedBy, b.Status)
	return err
}
func (d *DB) SuspendReservations(ctx context.Context, equipmentID string, w domain.TimeWindow) error {
	_, err := d.SQL.ExecContext(ctx, "UPDATE reservations SET status='suspended' WHERE equipment_id=? AND status IN ('held','checked_out') AND "+overlapsSQL(), equipmentID, w.End.Format(time.RFC3339Nano), w.Start.Format(time.RFC3339Nano))
	return err
}
