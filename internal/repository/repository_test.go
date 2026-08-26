package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"github.com/11DingKing/lab-scheduling/internal/repository"
	"github.com/11DingKing/lab-scheduling/internal/testutil"
	"testing"
	"time"
)

func window(t *testing.T, hours int) domain.TimeWindow {
	t.Helper()
	start := time.Now().UTC().Add(time.Hour)
	return domain.TimeWindow{Start: start, End: start.Add(time.Duration(hours) * time.Hour)}
}
func TestMigrationSeedsUsersEquipmentAndConsumables(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	var users int
	if err := db.SQL.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 4 {
		t.Fatalf("users=%d", users)
	}
	equipment, err := db.Equipment(ctx, "eq-cnc")
	if err != nil {
		t.Fatal(err)
	}
	if equipment.Status != domain.EquipmentReady {
		t.Fatal(equipment)
	}
	var available int
	if err := db.SQL.QueryRowContext(ctx, "SELECT available FROM consumables WHERE id='mat-al'").Scan(&available); err != nil {
		t.Fatal(err)
	}
	if available != 500 {
		t.Fatal(available)
	}
}
func TestMigrationIsIdempotent(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	var before int
	if err := db.SQL.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations").Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before != 1 {
		t.Fatal(before)
	}
}
func TestEquipmentVersionConflict(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	e, err := db.Equipment(ctx, "eq-cnc")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.UpdateEquipmentStatus(ctx, e.ID, domain.EquipmentMaintenance, e.Version); err != nil {
		t.Fatal(err)
	}
	if err = db.UpdateEquipmentStatus(ctx, e.ID, domain.EquipmentReady, e.Version); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err=%v", err)
	}
}
func TestEquipmentFiltering(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	if err := db.CreateEquipment(ctx, domain.Equipment{ID: "eq-lathe", Name: "Lathe", Kind: "lathe", Capacity: 8, Status: domain.EquipmentReady, Version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateEquipment(ctx, domain.Equipment{ID: "eq-off", Name: "Offline", Kind: "lathe", Capacity: 8, Status: domain.EquipmentOffline, Version: 1}); err != nil {
		t.Fatal(err)
	}
	ready, err := db.ListEquipment(ctx, string(domain.EquipmentReady))
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 2 {
		t.Fatalf("ready=%d", len(ready))
	}
	all, err := db.ListEquipment(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("all=%d", len(all))
	}
}
func TestBatchRoundTrip(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	w := window(t, 2)
	batch := domain.CourseBatch{ID: "batch-1", TeacherID: "u-teacher", Title: "Intro", StudentCount: 10, Qualification: "CNC-1", Status: "draft", Window: w}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatal(err)
	}
	got, err := db.Batch(ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != batch.Title || got.StudentCount != batch.StudentCount {
		t.Fatalf("got=%+v", got)
	}
	if !got.Window.Start.Equal(w.Start) {
		t.Fatalf("start %v %v", got.Window.Start, w.Start)
	}
}
func TestBookingAndReservationTransaction(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	w := window(t, 2)
	if err := db.CreateBatch(ctx, domain.CourseBatch{ID: "batch-2", TeacherID: "u-teacher", Title: "Batch", StudentCount: 4, Qualification: "CNC-1", Status: "draft", Window: w}); err != nil {
		t.Fatal(err)
	}
	req := domain.BookingRequest{ID: "req-1", BatchID: "batch-2", RequesterID: "u-teacher", EquipmentID: "eq-cnc", Window: w, Status: domain.RequestPending, Version: 1}
	if err := db.CreateBooking(ctx, req); err != nil {
		t.Fatal(err)
	}
	var res domain.Reservation
	err := db.WithTx(ctx, func(tx *sql.Tx) error {
		conflict, e := db.HasConflictTx(ctx, tx, "eq-cnc", w, "")
		if e != nil {
			return e
		}
		if conflict {
			return domain.ErrConflict
		}
		if e = db.SetBookingStatusTx(ctx, tx, req.ID, domain.RequestPending, domain.RequestApproved, 1); e != nil {
			return e
		}
		res = domain.Reservation{ID: "res-1", RequestID: req.ID, EquipmentID: "eq-cnc", HolderID: "u-teacher", Window: w, Status: domain.ReservationHeld, Version: 1}
		return db.CreateReservationTx(ctx, tx, res)
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := db.Reservation(ctx, res.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.ReservationHeld {
		t.Fatal(got.Status)
	}
}
func TestTransactionRollbackLeavesPending(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	w := window(t, 1)
	if err := db.CreateBatch(ctx, domain.CourseBatch{ID: "batch-3", TeacherID: "u-teacher", Title: "Batch", StudentCount: 4, Qualification: "CNC-1", Status: "draft", Window: w}); err != nil {
		t.Fatal(err)
	}
	req := domain.BookingRequest{ID: "req-rollback", BatchID: "batch-3", RequesterID: "u-teacher", EquipmentID: "eq-cnc", Window: w, Status: domain.RequestPending, Version: 1}
	if err := db.CreateBooking(ctx, req); err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("fail after status")
	if err := db.WithTx(ctx, func(tx *sql.Tx) error {
		if e := db.SetBookingStatusTx(ctx, tx, req.ID, domain.RequestPending, domain.RequestApproved, 1); e != nil {
			return e
		}
		return sentinel
	}); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	got, err := db.Booking(ctx, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.RequestPending {
		t.Fatalf("status leaked: %s", got.Status)
	}
}
func TestConflictAndMaintenanceQueries(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	w := window(t, 2)
	if err := db.CreateBatch(ctx, domain.CourseBatch{ID: "batch-4", TeacherID: "u-teacher", Title: "Batch", StudentCount: 4, Qualification: "CNC-1", Status: "draft", Window: w}); err != nil {
		t.Fatal(err)
	}
	req := domain.BookingRequest{ID: "req-conflict", BatchID: "batch-4", RequesterID: "u-teacher", EquipmentID: "eq-cnc", Window: w, Status: domain.RequestPending, Version: 1}
	if err := db.CreateBooking(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := db.WithTx(ctx, func(tx *sql.Tx) error {
		return db.CreateReservationTx(ctx, tx, domain.Reservation{ID: "res-conflict", RequestID: req.ID, EquipmentID: "eq-cnc", HolderID: "u-teacher", Window: w, Status: domain.ReservationHeld, Version: 1})
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateMaintenance(ctx, domain.MaintenanceBlock{ID: "maint-1", EquipmentID: "eq-cnc", Window: w, Reason: "inspection", CreatedBy: "u-admin", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	var conflict, blocked bool
	if err := db.WithTx(ctx, func(tx *sql.Tx) error {
		var e error
		conflict, e = db.HasConflictTx(ctx, tx, "eq-cnc", w, "")
		if e != nil {
			return e
		}
		blocked, e = db.HasMaintenanceTx(ctx, tx, "eq-cnc", w)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	if !conflict || !blocked {
		t.Fatalf("conflict=%v blocked=%v", conflict, blocked)
	}
}
func TestConsumableConsumeRestore(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	w := window(t, 1)
	if _, err := db.SQL.ExecContext(ctx, "INSERT INTO course_batches(id,teacher_id,title,student_count,qualification,status,starts_at,ends_at,created_at) VALUES('batch-consume','u-teacher','Consume',1,'CNC-1','draft',?,?,datetime('now'))", w.Start.Format(time.RFC3339Nano), w.End.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.ExecContext(ctx, "INSERT INTO booking_requests(id,batch_id,requester_id,equipment_id,starts_at,ends_at,status,version,created_at,updated_at) VALUES('req-consume','batch-consume','u-teacher','eq-cnc',?,?,?,1,datetime('now'),datetime('now'))", w.Start.Format(time.RFC3339Nano), w.End.Format(time.RFC3339Nano), domain.RequestApproved); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.ExecContext(ctx, "INSERT INTO reservations(id,request_id,equipment_id,holder_id,starts_at,ends_at,status,version,created_at) VALUES('res-consume','req-consume','eq-cnc','u-teacher',?,?,?,1,datetime('now'))", w.Start.Format(time.RFC3339Nano), w.End.Format(time.RFC3339Nano), domain.ReservationHeld); err != nil {
		t.Fatal(err)
	}
	if err := db.WithTx(ctx, func(tx *sql.Tx) error { return db.ConsumeTx(ctx, tx, "res-consume", "mat-al", 7) }); err != nil {
		t.Fatal(err)
	}
	var available int
	if err := db.SQL.QueryRowContext(ctx, "SELECT available FROM consumables WHERE id='mat-al'").Scan(&available); err != nil {
		t.Fatal(err)
	}
	if available != 493 {
		t.Fatal(available)
	}
	if err := db.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO booking_requests(id,batch_id,requester_id,equipment_id,starts_at,ends_at,status,version,created_at,updated_at) VALUES('req-restore','batch-consume','u-teacher','eq-cnc',?,?,?,1,datetime('now'),datetime('now'))", w.Start.Format(time.RFC3339Nano), w.End.Format(time.RFC3339Nano), domain.RequestApproved); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO reservations(id,request_id,equipment_id,holder_id,starts_at,ends_at,status,version,created_at) VALUES('res-restore','req-restore','eq-cnc','u-teacher',?,?,?,1,datetime('now'))", w.Start.Format(time.RFC3339Nano), w.End.Format(time.RFC3339Nano), domain.ReservationHeld); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO reservation_consumables(reservation_id,consumable_id,quantity) VALUES('res-restore','mat-al',7)")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.WithTx(ctx, func(tx *sql.Tx) error { return db.RestoreConsumablesTx(ctx, tx, "res-restore") }); err != nil {
		t.Fatal(err)
	}
	if err := db.SQL.QueryRowContext(ctx, "SELECT available FROM consumables WHERE id='mat-al'").Scan(&available); err != nil {
		t.Fatal(err)
	}
	if available != 500 {
		t.Fatal(available)
	}
}
func TestAuditTrailRoundTrip(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	now := time.Now().UTC()
	event := repository.AuditEvent{ID: "audit-1", ActorID: "u-admin", Action: "approve", EntityType: "reservation", EntityID: "res-1", Outcome: "ok", RequestID: "req-1", Details: "approved", CreatedAt: now}
	if err := db.AddAudit(ctx, event); err != nil {
		t.Fatal(err)
	}
	events, err := db.Audits(ctx, "reservation", "res-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Action != "approve" {
		t.Fatalf("events=%+v", events)
	}
}
func TestIncidentLifecycle(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	if _, err := db.SQL.ExecContext(ctx, "INSERT INTO course_batches(id,teacher_id,title,student_count,qualification,status,starts_at,ends_at,created_at) VALUES('batch-inc','u-teacher','Incident',1,'CNC-1','draft',datetime('now'),datetime('now','+1 hour'),datetime('now'))"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.ExecContext(ctx, "INSERT INTO booking_requests(id,batch_id,requester_id,equipment_id,starts_at,ends_at,status,version,created_at,updated_at) VALUES('req-inc','batch-inc','u-teacher','eq-cnc',datetime('now'),datetime('now','+1 hour'),?,1,datetime('now'),datetime('now'))", domain.RequestApproved); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.ExecContext(ctx, "INSERT INTO reservations(id,request_id,equipment_id,holder_id,starts_at,ends_at,status,version,created_at) VALUES('res-inc','req-inc','eq-cnc','u-teacher',datetime('now'),datetime('now','+1 hour'),?,1,datetime('now'))", domain.ReservationHeld); err != nil {
		t.Fatal(err)
	}
	i := repository.IncidentRecord{ID: "inc-1", ReservationID: "res-inc", EquipmentID: "eq-cnc", ReportedBy: "u-safety", Severity: "high", Description: "guard open", Status: "open", CreatedAt: time.Now().UTC()}
	if err := db.CreateIncident(ctx, i); err != nil {
		t.Fatal(err)
	}
	got, err := db.Incident(ctx, i.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Severity != "high" {
		t.Fatal(got)
	}
	if err := db.ResolveIncident(ctx, i.ID); err != nil {
		t.Fatal(err)
	}
	got, err = db.Incident(ctx, i.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "resolved" {
		t.Fatal(got.Status)
	}
}
