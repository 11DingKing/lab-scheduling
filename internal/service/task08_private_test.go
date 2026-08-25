package service_test

import (
	"context"
	"testing"
	"time"
	"github.com/11DingKing/lab-scheduling/internal/clock"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"github.com/11DingKing/lab-scheduling/internal/repository"
	"github.com/11DingKing/lab-scheduling/internal/service"
	"github.com/11DingKing/lab-scheduling/internal/testutil"
)

func TestIncidentFailureDoesNotSuspendReservation(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	now := time.Now().UTC()
	w := domain.TimeWindow{Start: now.Add(time.Hour), End: now.Add(2*time.Hour)}
	svc := service.Service{Store: db, Clock: clock.Real{}}
	teacher := domain.User{ID: "u-teacher", Role: domain.RoleTeacher}
	if err := svc.CreateBatch(ctx, teacher, domain.CourseBatch{ID: "batch-inc-fail", Title: "Incident", StudentCount: 1, Qualification: "CNC-1", Window: w}); err != nil { t.Fatal(err) }
	if _, err := svc.SubmitBooking(ctx, teacher, domain.BookingRequest{ID: "req-inc-fail", BatchID: "batch-inc-fail", EquipmentID: "eq-cnc", Window: w}, "", 0); err != nil { t.Fatal(err) }
	reservation, err := svc.ApproveBooking(ctx, domain.User{ID: "u-admin", Role: domain.RoleAdmin}, "req-inc-fail")
	if err != nil { t.Fatal(err) }
	if _, err := db.SQL.ExecContext(ctx, "CREATE TRIGGER fail_incident BEFORE INSERT ON incidents BEGIN SELECT RAISE(ABORT, 'incident blocked'); END"); err != nil { t.Fatal(err) }
	store := db
	err = service.ReportIncident(ctx, store, domain.User{ID: "u-safety", Role: domain.RoleSafety}, repository.IncidentRecord{ID: "inc-fail", ReservationID: reservation.ID, EquipmentID: "eq-cnc", Severity: "high", Description: "guard", Status: "open", CreatedAt: now}, domain.Equipment{ID: "eq-cnc"}, w, now.Format(time.RFC3339))
	if err == nil { t.Fatal("incident unexpectedly succeeded") }
	got, err := db.Reservation(ctx, reservation.ID)
	if err != nil { t.Fatal(err) }
	if got.Status != domain.ReservationHeld { t.Fatalf("reservation suspended without incident record: %s", got.Status) }
}
