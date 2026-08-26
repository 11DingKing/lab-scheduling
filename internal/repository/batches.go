package repository

import (
	"context"
	"database/sql"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"time"
)

func (d *DB) CreateBatch(ctx context.Context, b domain.CourseBatch) error {
	_, err := d.SQL.ExecContext(ctx, "INSERT INTO course_batches(id,teacher_id,title,student_count,qualification,status,starts_at,ends_at,created_at) VALUES(?,?,?,?,?,?,?,?,datetime('now'))", b.ID, b.TeacherID, b.Title, b.StudentCount, b.Qualification, b.Status, b.Window.Start.Format(time.RFC3339Nano), b.Window.End.Format(time.RFC3339Nano))
	return err
}
func (d *DB) Batch(ctx context.Context, id string) (domain.CourseBatch, error) {
	var b domain.CourseBatch
	var start, end string
	err := d.SQL.QueryRowContext(ctx, "SELECT id,teacher_id,title,student_count,qualification,status,starts_at,ends_at FROM course_batches WHERE id=?", id).Scan(&b.ID, &b.TeacherID, &b.Title, &b.StudentCount, &b.Qualification, &b.Status, &start, &end)
	if err == sql.ErrNoRows {
		return b, domain.ErrNotFound
	}
	if err != nil {
		return b, err
	}
	b.Window.Start, _ = time.Parse(time.RFC3339Nano, start)
	b.Window.End, _ = time.Parse(time.RFC3339Nano, end)
	return b, nil
}
