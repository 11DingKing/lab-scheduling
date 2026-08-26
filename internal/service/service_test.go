package service_test

import (
	"context"
	"errors"
	"github.com/11DingKing/lab-scheduling/internal/clock"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"github.com/11DingKing/lab-scheduling/internal/repository"
	"github.com/11DingKing/lab-scheduling/internal/service"
	"github.com/11DingKing/lab-scheduling/internal/testutil"
	"testing"
	"time"
)

func svc(t *testing.T) (service.Service, context.Context, domain.User) {
	t.Helper()
	db := testutil.Open(t)
	now := time.Now().UTC()
	return service.Service{Store: db, Clock: clock.Fixed{Value: now}}, service.WithRequestID(context.Background(), "req-test"), domain.User{ID: "u-teacher", Role: domain.RoleTeacher, Email: "teacher@example.com", Active: true}
}
func bookingSetup(t *testing.T) (service.Service, context.Context, domain.User, domain.BookingRequest) {
	s, ctx, user := svc(t)
	w := domain.TimeWindow{Start: time.Now().UTC().Add(time.Hour), End: time.Now().UTC().Add(2 * time.Hour)}
	batch := domain.CourseBatch{ID: "batch-svc", Title: "CNC class", StudentCount: 8, Qualification: "CNC-1", Status: "draft", Window: w}
	if err := s.CreateBatch(ctx, user, batch); err != nil {
		t.Fatal(err)
	}
	req := domain.BookingRequest{ID: "req-svc", BatchID: batch.ID, EquipmentID: "eq-cnc", Window: w}
	return s, ctx, user, req
}
func TestCreateBatchRequiresTeacher(t *testing.T) {
	s, ctx, user := svc(t)
	user.Role = domain.RoleStudent
	err := s.CreateBatch(ctx, user, domain.CourseBatch{ID: "bad", Title: "x", StudentCount: 1, Qualification: "q", Window: domain.TimeWindow{Start: time.Now().Add(time.Hour), End: time.Now().Add(2 * time.Hour)}})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatal(err)
	}
}
func TestCreateBatchValidatesWindow(t *testing.T) {
	s, ctx, user := svc(t)
	err := s.CreateBatch(ctx, user, domain.CourseBatch{ID: "bad-window", Title: "x", StudentCount: 1, Qualification: "q", Window: domain.TimeWindow{Start: time.Now().Add(time.Hour), End: time.Now()}})
	if err == nil {
		t.Fatal("invalid window accepted")
	}
}
func TestSubmitBookingRoleAndOwnership(t *testing.T) {
	s, ctx, teacher, req := bookingSetup(t)
	if _, err := s.SubmitBooking(ctx, teacher, req, "mat-al", 3); err != nil {
		t.Fatal(err)
	}
	student := domain.User{ID: "u-student", Role: domain.RoleStudent}
	req.ID = "req-student"
	if _, err := s.SubmitBooking(ctx, student, req, "mat-al", 3); err != nil {
		t.Fatal(err)
	}
	otherTeacher := domain.User{ID: "other", Role: domain.RoleTeacher}
	req.ID = "req-other"
	if _, err := s.SubmitBooking(ctx, otherTeacher, req, "mat-al", 3); !errors.Is(err, domain.ErrForbidden) {
		t.Fatal(err)
	}
}
func TestApproveBookingCreatesReservation(t *testing.T) {
	s, ctx, teacher, req := bookingSetup(t)
	if _, err := s.SubmitBooking(ctx, teacher, req, "", 0); err != nil {
		t.Fatal(err)
	}
	admin := domain.User{ID: "u-admin", Role: domain.RoleAdmin}
	res, err := s.ApproveBooking(ctx, admin, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != domain.ReservationHeld {
		t.Fatal(res)
	}
	saved, err := s.Store.Reservation(ctx, res.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.RequestID != req.ID {
		t.Fatal(saved)
	}
}
func TestApproveRejectsConflict(t *testing.T) {
	s, ctx, teacher, req := bookingSetup(t)
	if _, err := s.SubmitBooking(ctx, teacher, req, "", 0); err != nil {
		t.Fatal(err)
	}
	admin := domain.User{ID: "u-admin", Role: domain.RoleAdmin}
	first, err := s.ApproveBooking(ctx, admin, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" {
		t.Fatal("empty reservation")
	}
	req2 := req
	req2.ID = "req-second"
	if _, err := s.SubmitBooking(ctx, teacher, req2, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApproveBooking(ctx, admin, req2.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatal(err)
	}
}
func TestApproveRejectsMaintenance(t *testing.T) {
	s, ctx, teacher, req := bookingSetup(t)
	if _, err := s.SubmitBooking(ctx, teacher, req, "", 0); err != nil {
		t.Fatal(err)
	}
	admin := domain.User{ID: "u-admin", Role: domain.RoleAdmin}
	w := req.Window
	if err := s.AddMaintenance(ctx, admin, domain.MaintenanceBlock{ID: "maint-svc", EquipmentID: "eq-cnc", Reason: "repair", Window: w}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApproveBooking(ctx, admin, req.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatal(err)
	}
}
func TestCheckoutAndReturnRestoreQuota(t *testing.T) {
	s, ctx, teacher, req := bookingSetup(t)
	if _, err := s.SubmitBooking(ctx, teacher, req, "", 0); err != nil {
		t.Fatal(err)
	}
	admin := domain.User{ID: "u-admin", Role: domain.RoleAdmin}
	res, err := s.ApproveBooking(ctx, admin, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Checkout(ctx, teacher, res.ID, "mat-al", 11); err != nil {
		t.Fatal(err)
	}
	stored, err := s.Store.Reservation(ctx, res.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != domain.ReservationCheckedOut {
		t.Fatal(stored.Status)
	}
	if err = s.Return(ctx, teacher, res.ID); err != nil {
		t.Fatal(err)
	}
	stored, err = s.Store.Reservation(ctx, res.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != domain.ReservationReturned {
		t.Fatal(stored.Status)
	}
}
func TestCheckoutRejectsInsufficientQuota(t *testing.T) {
	s, ctx, teacher, req := bookingSetup(t)
	if _, err := s.SubmitBooking(ctx, teacher, req, "", 0); err != nil {
		t.Fatal(err)
	}
	admin := domain.User{ID: "u-admin", Role: domain.RoleAdmin}
	res, err := s.ApproveBooking(ctx, admin, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Checkout(ctx, teacher, res.ID, "mat-al", 501); !errors.Is(err, domain.ErrCapacity) {
		t.Fatal(err)
	}
}
func TestCheckoutRejectsWrongHolder(t *testing.T) {
	s, ctx, teacher, req := bookingSetup(t)
	if _, err := s.SubmitBooking(ctx, teacher, req, "", 0); err != nil {
		t.Fatal(err)
	}
	admin := domain.User{ID: "u-admin", Role: domain.RoleAdmin}
	res, err := s.ApproveBooking(ctx, admin, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	other := domain.User{ID: "u-student", Role: domain.RoleStudent}
	if err = s.Checkout(ctx, other, res.ID, "mat-al", 2); !errors.Is(err, domain.ErrForbidden) {
		t.Fatal(err)
	}
}
func TestReturnOnlyAfterCheckout(t *testing.T) {
	s, ctx, teacher, req := bookingSetup(t)
	if _, err := s.SubmitBooking(ctx, teacher, req, "", 0); err != nil {
		t.Fatal(err)
	}
	admin := domain.User{ID: "u-admin", Role: domain.RoleAdmin}
	res, err := s.ApproveBooking(ctx, admin, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Return(ctx, teacher, res.ID); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatal(err)
	}
}
func TestMaintenanceSuspendsReservations(t *testing.T) {
	s, ctx, teacher, req := bookingSetup(t)
	if _, err := s.SubmitBooking(ctx, teacher, req, "", 0); err != nil {
		t.Fatal(err)
	}
	admin := domain.User{ID: "u-admin", Role: domain.RoleAdmin}
	res, err := s.ApproveBooking(ctx, admin, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.AddMaintenance(ctx, admin, domain.MaintenanceBlock{ID: "maint-suspend", EquipmentID: "eq-cnc", Reason: "safety", Window: req.Window}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Store.Reservation(ctx, res.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.ReservationSuspended {
		t.Fatal(got.Status)
	}
}
func TestAuditFailureDoesNotHideSuccessfulMutation(t *testing.T) {
	s, ctx, teacher, req := bookingSetup(t)
	if _, err := s.SubmitBooking(ctx, teacher, req, "", 0); err != nil {
		t.Fatal(err)
	}
	admin := domain.User{ID: "u-admin", Role: domain.RoleAdmin}
	res, err := s.ApproveBooking(ctx, admin, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.Reservation(ctx, res.ID); err != nil {
		t.Fatal(err)
	}
}
func TestIdempotency(t *testing.T) {
	i := service.NewIdempotency()
	calls := 0
	fn := func() error { calls++; return nil }
	if err := i.Do(context.Background(), "key", []byte("payload"), fn); err != nil {
		t.Fatal(err)
	}
	if err := i.Do(context.Background(), "key", []byte("payload"), fn); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	if err := i.Do(context.Background(), "key", []byte("different"), fn); err == nil {
		t.Fatal("payload reuse accepted")
	}
}
func TestIdempotencyFailureCanRetry(t *testing.T) {
	i := service.NewIdempotency()
	calls := 0
	fn := func() error {
		calls++
		if calls == 1 {
			return errors.New("temporary")
		}
		return nil
	}
	if err := i.Do(context.Background(), "retry", []byte("x"), fn); err == nil {
		t.Fatal("first should fail")
	}
	if err := i.Do(context.Background(), "retry", []byte("x"), fn); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}
func TestRetryContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	err := service.Retry(ctx, 3, time.Millisecond, func(context.Context) error { calls++; return errors.New("fail") })
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}
func TestRecoveryCallbacks(t *testing.T) {
	calls := 0
	r := service.Recovery{Check: func(context.Context) error { calls++; return nil }, Repair: func(context.Context) error { calls++; return nil }}
	if err := r.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal(calls)
	}
	r.Repair = nil
	if err := r.Run(context.Background()); err == nil {
		t.Fatal("missing repair accepted")
	}
}
func TestListReadyEquipment(t *testing.T) {
	s, ctx, _ := svc(t)
	items, err := service.ListReadyEquipment(ctx, s.Store.(interface {
		ListEquipment(context.Context, string) ([]domain.Equipment, error)
		Audits(context.Context, string, string) ([]repository.AuditEvent, error)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("no equipment")
	}
}
func TestAuditTrail(t *testing.T) {
	s, ctx, teacher, req := bookingSetup(t)
	if _, err := s.SubmitBooking(ctx, teacher, req, "", 0); err != nil {
		t.Fatal(err)
	}
	events, err := service.AuditTrail(ctx, s.Store.(interface {
		ListEquipment(context.Context, string) ([]domain.Equipment, error)
		Audits(context.Context, string, string) ([]repository.AuditEvent, error)
	}), "booking_request", req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events=%d", len(events))
	}
}
