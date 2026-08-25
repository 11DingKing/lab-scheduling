package domain

import (
	"fmt"
	"time"
)

type Permission string

const (
	PermissionCreateBatch       Permission = "create_batch"
	PermissionSubmitBooking     Permission = "submit_booking"
	PermissionApprove           Permission = "approve"
	PermissionCheckOut          Permission = "check_out"
	PermissionReportIncident    Permission = "report_incident"
	PermissionManageMaintenance Permission = "manage_maintenance"
)

func Allowed(role Role, p Permission) bool {
	switch role {
	case RoleTeacher:
		return p == PermissionCreateBatch || p == PermissionSubmitBooking
	case RoleStudent:
		return p == PermissionSubmitBooking || p == PermissionCheckOut
	case RoleAdmin:
		return true
	case RoleSafety:
		return p == PermissionReportIncident || p == PermissionManageMaintenance
	default:
		return false
	}
}
func ValidateQualification(required string, provided []string) error {
	for _, v := range provided {
		if v == required {
			return nil
		}
	}
	return fmt.Errorf("qualification %q is missing", required)
}
func InSafetyWindow(now time.Time, w TimeWindow) bool {
	return !now.Before(w.Start) && now.Before(w.End)
}
func IsCrossDay(w TimeWindow) bool { return w.Start.UTC().YearDay() != w.End.UTC().YearDay() }
