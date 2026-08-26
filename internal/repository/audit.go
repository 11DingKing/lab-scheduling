package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type AuditEvent struct {
	ID, ActorID, Action, EntityType, EntityID, Outcome, RequestID, Details string
	CreatedAt                                                              time.Time
}

func (d *DB) AddAudit(ctx context.Context, a AuditEvent) error {
	_, err := d.SQL.ExecContext(ctx, "INSERT INTO audit_events(id,actor_id,action,entity_type,entity_id,outcome,request_id,details,created_at) VALUES(?,?,?,?,?,?,?,?,?)", a.ID, a.ActorID, a.Action, a.EntityType, a.EntityID, a.Outcome, a.RequestID, a.Details, a.CreatedAt.Format(time.RFC3339Nano))
	return err
}
func (d *DB) Audits(ctx context.Context, entityType, entityID string) ([]AuditEvent, error) {
	rows, err := d.SQL.QueryContext(ctx, "SELECT id,actor_id,action,entity_type,entity_id,outcome,request_id,details,created_at FROM audit_events WHERE entity_type=? AND entity_id=? ORDER BY created_at", entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEvent{}
	for rows.Next() {
		var a AuditEvent
		var actor sql.NullString
		var when string
		if err := rows.Scan(&a.ID, &actor, &a.Action, &a.EntityType, &a.EntityID, &a.Outcome, &a.RequestID, &a.Details, &when); err != nil {
			return nil, err
		}
		a.ActorID = actor.String
		a.CreatedAt, _ = time.Parse(time.RFC3339Nano, when)
		out = append(out, a)
	}
	return out, rows.Err()
}
func newID(prefix string, now time.Time) string { return fmt.Sprintf("%s-%d", prefix, now.UnixNano()) }
