package repository

import (
	"context"
	"database/sql"
	"time"
)

type IncidentRecord struct {
	ID, ReservationID, EquipmentID, ReportedBy, Severity, Description, Status string
	CreatedAt                                                                 time.Time
}

func (d *DB) CreateIncident(ctx context.Context, i IncidentRecord) error {
	_, err := d.SQL.ExecContext(ctx, "INSERT INTO incidents(id,reservation_id,equipment_id,reported_by,severity,description,status,created_at) VALUES(?,?,?,?,?,?,?,?)", i.ID, i.ReservationID, i.EquipmentID, i.ReportedBy, i.Severity, i.Description, i.Status, i.CreatedAt.Format(time.RFC3339Nano))
	return err
}
func (d *DB) Incident(ctx context.Context, id string) (IncidentRecord, error) {
	var i IncidentRecord
	var when string
	err := d.SQL.QueryRowContext(ctx, "SELECT id,reservation_id,equipment_id,reported_by,severity,description,status,created_at FROM incidents WHERE id=?", id).Scan(&i.ID, &i.ReservationID, &i.EquipmentID, &i.ReportedBy, &i.Severity, &i.Description, &i.Status, &when)
	if err == sql.ErrNoRows {
		return i, sql.ErrNoRows
	}
	if err != nil {
		return i, err
	}
	i.CreatedAt, _ = time.Parse(time.RFC3339Nano, when)
	return i, nil
}
func (d *DB) ResolveIncident(ctx context.Context, id string) error {
	_, err := d.SQL.ExecContext(ctx, "UPDATE incidents SET status='resolved',resolved_at=datetime('now') WHERE id=? AND status<>'resolved'", id)
	return err
}
