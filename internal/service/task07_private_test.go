package service_test

import (
	"context"
	"testing"
	"time"
	"github.com/11DingKing/lab-scheduling/internal/clock"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"github.com/11DingKing/lab-scheduling/internal/service"
	"github.com/11DingKing/lab-scheduling/internal/testutil"
)

func TestMaintenanceFailureDoesNotSuspendReservation(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	now := time.Now().UTC()
	w := domain.TimeWindow{Start: now.Add(time.Hour), End: now.Add(2*time.Hour)}
	svc := service.Service{Store: db, Clock: clock.Real{}}
	teacher := domain.User{ID: "u-teacher", Role: domain.RoleTeacher}
	if err := svc.CreateBatch(ctx, teacher, domain.CourseBatch{ID: "batch-maint", Title: "Maintenance", StudentCount: 1, Qualification: "CNC-1", Window: w}); err != nil { t.Fatal(err) }
	if _, err := svc.SubmitBooking(ctx, teacher, domain.BookingRequest{ID: "req-maint", BatchID: "batch-maint", EquipmentID: "eq-cnc", Window: w}, "", 0); err != nil { t.Fatal(err) }
	reservation, err := svc.ApproveBooking(ctx, domain.User{ID: "u-admin", Role: domain.RoleAdmin}, "req-maint")
	if err != nil { t.Fatal(err) }
	if _, err := db.SQL.ExecContext(ctx, "CREATE TRIGGER fail_maintenance BEFORE INSERT ON maintenance_blocks BEGIN SELECT RAISE(ABORT, 'maintenance blocked'); END"); err != nil { t.Fatal(err) }
	if err := svc.AddMaintenance(ctx, domain.User{ID: "u-admin", Role: domain.RoleAdmin}, domain.MaintenanceBlock{ID: "maint-fail", EquipmentID: "eq-cnc", Reason: "repair", Window: w}); err == nil { t.Fatal("maintenance unexpectedly succeeded") }
	got, err := db.Reservation(ctx, reservation.ID)
	if err != nil { t.Fatal(err) }
	if got.Status != domain.ReservationHeld { t.Fatalf("reservation suspended despite failed maintenance insert: %s", got.Status) }
}
