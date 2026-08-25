package service

import "github.com/11DingKing/lab-scheduling/internal/domain"

func CanReadReservation(actor domain.User, r domain.Reservation) bool {
	return actor.Role == domain.RoleAdmin || actor.Role == domain.RoleSafety || actor.ID == r.HolderID
}
func CanCancelRequest(actor domain.User, r domain.BookingRequest, b domain.CourseBatch) bool {
	if actor.Role == domain.RoleAdmin {
		return true
	}
	return actor.Role == domain.RoleTeacher && actor.ID == b.TeacherID && r.Status == domain.RequestPending
}
func CanManageEquipment(actor domain.User) bool {
	return actor.Role == domain.RoleAdmin || actor.Role == domain.RoleSafety
}
func IsPrivileged(actor domain.User) bool {
	return actor.Role == domain.RoleAdmin || actor.Role == domain.RoleSafety
}
func IsLearner(actor domain.User) bool { return actor.Role == domain.RoleStudent }
