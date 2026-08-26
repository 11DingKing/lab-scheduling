package repository

import (
	"context"
	"database/sql"
	"time"
)

type IdempotencyRecord struct {
	Key, ActorID, RequestHash string
	Code                      int
	Body                      string
	ExpiresAt                 time.Time
}

func (d *DB) FindIdempotency(ctx context.Context, key, actor string) (IdempotencyRecord, error) {
	var r IdempotencyRecord
	var created, expires string
	err := d.SQL.QueryRowContext(ctx, "SELECT key,actor_id,request_hash,response_code,response_body,created_at,expires_at FROM idempotency_records WHERE key=? AND actor_id=?", key, actor).Scan(&r.Key, &r.ActorID, &r.RequestHash, &r.Code, &r.Body, &created, &expires)
	if err == sql.ErrNoRows {
		return r, sql.ErrNoRows
	}
	if err != nil {
		return r, err
	}
	r.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
	return r, nil
}
func (d *DB) SaveIdempotency(ctx context.Context, r IdempotencyRecord) error {
	_, err := d.SQL.ExecContext(ctx, "INSERT INTO idempotency_records(key,actor_id,request_hash,response_code,response_body,created_at,expires_at) VALUES(?,?,?,?,?,datetime('now'),?)", r.Key, r.ActorID, r.RequestHash, r.Code, r.Body, r.ExpiresAt.Format(time.RFC3339Nano))
	return err
}
func (d *DB) PurgeIdempotency(ctx context.Context, now time.Time) (int64, error) {
	res, err := d.SQL.ExecContext(ctx, "DELETE FROM idempotency_records WHERE expires_at<?", now.Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
