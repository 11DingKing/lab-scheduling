package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("resource not found")
	ErrForbidden    = errors.New("operation forbidden")
	ErrConflict     = errors.New("resource conflict")
	ErrInvalidState = errors.New("invalid state transition")
	ErrCapacity     = errors.New("capacity exceeded")
	ErrExpired      = errors.New("session expired")
)

type Role string

const (
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
	RoleAdmin   Role = "admin"
	RoleSafety  Role = "safety"
)

func (r Role) Valid() bool {
	return r == RoleTeacher || r == RoleStudent || r == RoleAdmin || r == RoleSafety
}

type RequestStatus string

const (
	RequestPending   RequestStatus = "pending"
	RequestApproved  RequestStatus = "approved"
	RequestRejected  RequestStatus = "rejected"
	RequestCancelled RequestStatus = "cancelled"
)

func (s RequestStatus) CanMove(to RequestStatus) bool {
	switch s {
	case RequestPending:
		return to == RequestApproved || to == RequestRejected || to == RequestCancelled
	case RequestApproved:
		return to == RequestCancelled
	default:
		return false
	}
}

type ReservationStatus string

const (
	ReservationHeld       ReservationStatus = "held"
	ReservationCheckedOut ReservationStatus = "checked_out"
	ReservationReturned   ReservationStatus = "returned"
	ReservationSuspended  ReservationStatus = "suspended"
	ReservationNoShow     ReservationStatus = "no_show"
)

func (s ReservationStatus) CanMove(to ReservationStatus) bool {
	switch s {
	case ReservationHeld:
		return to == ReservationCheckedOut || to == ReservationCancelled || to == ReservationNoShow || to == ReservationSuspended
	case ReservationCheckedOut:
		return to == ReservationReturned || to == ReservationSuspended
	case ReservationSuspended:
		return to == ReservationReturned || to == ReservationCancelled
	default:
		return false
	}
}

const ReservationCancelled ReservationStatus = "cancelled"

type EquipmentStatus string

const (
	EquipmentReady       EquipmentStatus = "ready"
	EquipmentMaintenance EquipmentStatus = "maintenance"
	EquipmentOffline     EquipmentStatus = "offline"
)

type TimeWindow struct{ Start, End time.Time }

func NewWindow(start, end time.Time) (TimeWindow, error) {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return TimeWindow{}, fmt.Errorf("invalid time window: %w", ErrConflict)
	}
	return TimeWindow{start.UTC(), end.UTC()}, nil
}
func (w TimeWindow) Overlaps(other TimeWindow) bool {
	return w.Start.Before(other.End) && other.Start.Before(w.End)
}

type User struct {
	ID, Email, DisplayName string
	Role                   Role
	Active                 bool
}
type Equipment struct {
	ID, Name, Kind string
	Capacity       int
	Status         EquipmentStatus
	Version        int
}
type CourseBatch struct {
	ID, TeacherID, Title, Qualification string
	StudentCount                        int
	Status                              string
	Window                              TimeWindow
}
type BookingRequest struct {
	ID, BatchID, RequesterID, EquipmentID string
	Window                                TimeWindow
	Status                                RequestStatus
	Version                               int
}
type Reservation struct {
	ID, RequestID, EquipmentID, HolderID string
	Window                               TimeWindow
	Status                               ReservationStatus
	Version                              int
}
type Consumable struct {
	ID, Name, Unit            string
	Quota, Available, Version int
}
type MaintenanceBlock struct {
	ID, EquipmentID, Reason, CreatedBy string
	Window                             TimeWindow
	Status                             string
}
type Incident struct {
	ID, ReservationID, EquipmentID, ReportedBy, Severity, Description, Status string
	CreatedAt                                                                 time.Time
}

func NormalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
func RequireRole(actual Role, allowed ...Role) error {
	for _, role := range allowed {
		if actual == role {
			return nil
		}
	}
	return ErrForbidden
}
