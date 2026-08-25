package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/11DingKing/lab-scheduling/internal/clock"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"github.com/11DingKing/lab-scheduling/internal/repository"
	"sync/atomic"
)

type Store interface {
	CreateBatch(context.Context, domain.CourseBatch) error
	Batch(context.Context, string) (domain.CourseBatch, error)
	Equipment(context.Context, string) (domain.Equipment, error)
	CreateBooking(context.Context, domain.BookingRequest) error
	Booking(context.Context, string) (domain.BookingRequest, error)
	WithTx(context.Context, func(*sql.Tx) error) error
	SetBookingStatusTx(context.Context, *sql.Tx, string, domain.RequestStatus, domain.RequestStatus, int) error
	HasConflictTx(context.Context, *sql.Tx, string, domain.TimeWindow, string) (bool, error)
	HasMaintenanceTx(context.Context, *sql.Tx, string, domain.TimeWindow) (bool, error)
	CreateReservationTx(context.Context, *sql.Tx, domain.Reservation) error
	Reservation(context.Context, string) (domain.Reservation, error)
	SetReservationStatus(context.Context, string, domain.ReservationStatus, domain.ReservationStatus, int) error
	SetReservationStatusTx(context.Context, *sql.Tx, string, domain.ReservationStatus, domain.ReservationStatus, int) error
	AddAudit(context.Context, repository.AuditEvent) error
	CreateMaintenance(context.Context, domain.MaintenanceBlock) error
	SuspendReservations(context.Context, string, domain.TimeWindow) error
	ConsumeTx(context.Context, *sql.Tx, string, string, int) error
	RestoreConsumablesTx(context.Context, *sql.Tx, string) error
}
type Service struct {
	Store Store
	Clock clock.Clock
}

var auditSequence atomic.Uint64

func (s Service) audit(ctx context.Context, actor, action, entity, id, outcome, request string) error {
	sequence := auditSequence.Add(1)
	return s.Store.AddAudit(ctx, repository.AuditEvent{ID: fmt.Sprintf("audit-%d-%d", s.Clock.Now().UnixNano(), sequence), ActorID: actor, Action: action, EntityType: entity, EntityID: id, Outcome: outcome, RequestID: request, Details: outcome, CreatedAt: s.Clock.Now()})
}
func (s Service) CreateBatch(ctx context.Context, teacher domain.User, b domain.CourseBatch) error {
	if err := domain.RequireRole(teacher.Role, domain.RoleTeacher, domain.RoleAdmin); err != nil {
		return err
	}
	b.TeacherID = teacher.ID
	if err := domain.ValidateBatch(b, s.Clock.Now()); err != nil {
		return err
	}
	if b.Status == "" {
		b.Status = "draft"
	}
	if err := s.Store.CreateBatch(ctx, b); err != nil {
		return err
	}
	return s.audit(ctx, teacher.ID, "batch.create", "course_batch", b.ID, "ok", requestID(ctx))
}
func requestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestKey{}).(string); ok {
		return v
	}
	return "system"
}

type requestKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestKey{}, id)
}

func (s Service) SubmitBooking(ctx context.Context, actor domain.User, r domain.BookingRequest, consumableID string, quantity int) (domain.Reservation, error) {
	if err := domain.RequireRole(actor.Role, domain.RoleTeacher, domain.RoleStudent); err != nil {
		return domain.Reservation{}, err
	}
	if quantity < 0 {
		return domain.Reservation{}, fmt.Errorf("invalid quantity")
	}
	batch, err := s.Store.Batch(ctx, r.BatchID)
	if err != nil {
		return domain.Reservation{}, err
	}
	if actor.Role == domain.RoleTeacher && batch.TeacherID != actor.ID {
		return domain.Reservation{}, domain.ErrForbidden
	}
	equip, err := s.Store.Equipment(ctx, r.EquipmentID)
	if err != nil {
		return domain.Reservation{}, err
	}
	if equip.Status != domain.EquipmentReady {
		return domain.Reservation{}, fmt.Errorf("equipment unavailable: %w", domain.ErrConflict)
	}
	r.RequesterID = actor.ID
	r.Status = domain.RequestPending
	r.Version = 1
	if err := s.Store.CreateBooking(ctx, r); err != nil {
		return domain.Reservation{}, err
	}
	return domain.Reservation{}, s.audit(ctx, actor.ID, "booking.submit", "booking_request", r.ID, "pending", requestID(ctx))
}
func (s Service) ApproveBooking(ctx context.Context, admin domain.User, id string) (domain.Reservation, error) {
	if err := domain.RequireRole(admin.Role, domain.RoleAdmin); err != nil {
		return domain.Reservation{}, err
	}
	req, err := s.Store.Booking(ctx, id)
	if err != nil {
		return domain.Reservation{}, err
	}
	batch, err := s.Store.Batch(ctx, req.BatchID)
	if err != nil {
		return domain.Reservation{}, err
	}
	if batch.StudentCount <= 0 {
		return domain.Reservation{}, domain.ErrCapacity
	}
	var result domain.Reservation
	err = s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		conflict, e := s.Store.HasConflictTx(ctx, tx, req.EquipmentID, req.Window, "")
		if e != nil {
			return e
		}
		if conflict {
			return domain.ErrConflict
		}
		blocked, e := s.Store.HasMaintenanceTx(ctx, tx, req.EquipmentID, req.Window)
		if e != nil {
			return e
		}
		if blocked {
			return fmt.Errorf("maintenance window: %w", domain.ErrConflict)
		}
		if e = s.Store.SetBookingStatusTx(ctx, tx, id, domain.RequestPending, domain.RequestApproved, req.Version); e != nil {
			return e
		}
		result = domain.Reservation{ID: fmt.Sprintf("res-%d", s.Clock.Now().UnixNano()), RequestID: id, EquipmentID: req.EquipmentID, HolderID: req.RequesterID, Window: req.Window, Status: domain.ReservationHeld, Version: 1}
		return s.Store.CreateReservationTx(ctx, tx, result)
	})
	if err != nil {
		return domain.Reservation{}, err
	}
	if err = s.audit(ctx, admin.ID, "booking.approve", "reservation", result.ID, "ok", requestID(ctx)); err != nil {
		return domain.Reservation{}, fmt.Errorf("audit approval: %w", err)
	}
	return result, nil
}
func (s Service) Checkout(ctx context.Context, actor domain.User, id string, consumableID string, quantity int) error {
	r, err := s.Store.Reservation(ctx, id)
	if err != nil {
		return err
	}
	if actor.ID != r.HolderID && actor.Role != domain.RoleAdmin {
		return domain.ErrForbidden
	}
	if !r.Status.CanMove(domain.ReservationCheckedOut) {
		return domain.ErrInvalidState
	}
	if err = domain.ValidateQuantity(quantity); err != nil {
		return err
	}
	err = s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		if e := s.Store.ConsumeTx(ctx, tx, id, consumableID, quantity); e != nil {
			return e
		}
		return s.Store.SetReservationStatusTx(ctx, tx, id, domain.ReservationHeld, domain.ReservationCheckedOut, r.Version)
	})
	if err != nil {
		return err
	}
	return s.audit(ctx, actor.ID, "reservation.checkout", "reservation", id, "ok", requestID(ctx))
}
func (s Service) Return(ctx context.Context, actor domain.User, id string) error {
	r, err := s.Store.Reservation(ctx, id)
	if err != nil {
		return err
	}
	if actor.ID != r.HolderID && actor.Role != domain.RoleAdmin {
		return domain.ErrForbidden
	}
	if !r.Status.CanMove(domain.ReservationReturned) {
		return domain.ErrInvalidState
	}
	err = s.Store.WithTx(ctx, func(tx *sql.Tx) error {
		if e := s.Store.RestoreConsumablesTx(ctx, tx, id); e != nil {
			return e
		}
		return s.Store.SetReservationStatusTx(ctx, tx, id, r.Status, domain.ReservationReturned, r.Version)
	})
	if err != nil {
		return err
	}
	return s.audit(ctx, actor.ID, "reservation.return", "reservation", id, "ok", requestID(ctx))
}
func (s Service) AddMaintenance(ctx context.Context, actor domain.User, b domain.MaintenanceBlock) error {
	if err := domain.RequireRole(actor.Role, domain.RoleAdmin, domain.RoleSafety); err != nil {
		return err
	}
	if b.Window.End.Before(s.Clock.Now()) {
		return errors.New("maintenance window has ended")
	}
	b.CreatedBy = actor.ID
	b.Status = "active"
	if err := s.Store.SuspendReservations(ctx, b.EquipmentID, b.Window); err != nil {
		return err
	}
	if err := s.Store.CreateMaintenance(ctx, b); err != nil {
		return err
	}
	return s.audit(ctx, actor.ID, "maintenance.create", "equipment", b.EquipmentID, "ok", requestID(ctx))
}
