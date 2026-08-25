package service_test

import (
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"github.com/11DingKing/lab-scheduling/internal/service"
	"testing"
)

func TestReservationVisibility(t *testing.T) {
	r := domain.Reservation{HolderID: "student"}
	cases := []struct {
		role, id string
		want     bool
	}{{"admin", "other", true}, {"safety", "other", true}, {"student", "student", true}, {"student", "other", false}}
	for _, tc := range cases {
		if got := service.CanReadReservation(domain.User{ID: tc.id, Role: domain.Role(tc.role)}, r); got != tc.want {
			t.Fatalf("%+v got %v", tc, got)
		}
	}
}
func TestCancelAuthorization(t *testing.T) {
	b := domain.CourseBatch{TeacherID: "teacher"}
	r := domain.BookingRequest{RequesterID: "teacher", Status: domain.RequestPending}
	if !service.CanCancelRequest(domain.User{ID: "teacher", Role: domain.RoleTeacher}, r, b) {
		t.Fatal("teacher denied")
	}
	if service.CanCancelRequest(domain.User{ID: "student", Role: domain.RoleStudent}, r, b) {
		t.Fatal("student allowed")
	}
	r.Status = domain.RequestApproved
	if service.CanCancelRequest(domain.User{ID: "teacher", Role: domain.RoleTeacher}, r, b) {
		t.Fatal("approved cancel allowed")
	}
	if !service.CanCancelRequest(domain.User{Role: domain.RoleAdmin}, r, b) {
		t.Fatal("admin denied")
	}
}
func TestEquipmentManagementRoles(t *testing.T) {
	if !service.CanManageEquipment(domain.User{Role: domain.RoleAdmin}) {
		t.Fatal("admin denied")
	}
	if !service.CanManageEquipment(domain.User{Role: domain.RoleSafety}) {
		t.Fatal("safety denied")
	}
	if service.CanManageEquipment(domain.User{Role: domain.RoleTeacher}) {
		t.Fatal("teacher allowed")
	}
}
