package domain_test

import (
	"errors"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"testing"
	"time"
)

func TestTimeWindowValidation(t *testing.T) {
	base := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		start, end time.Time
		ok         bool
	}{{"valid", base, base.Add(time.Hour), true}, {"equal", base, base, false}, {"reverse", base.Add(time.Hour), base, false}, {"zero", time.Time{}, base, false}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewWindow(tc.start, tc.end)
			if (tc.ok && err != nil) || (!tc.ok && err == nil) {
				t.Fatalf("window result err=%v", err)
			}
		})
	}
}
func TestOverlapBoundaries(t *testing.T) {
	base := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	left, _ := domain.NewWindow(base, base.Add(time.Hour))
	tests := []struct {
		name    string
		other   domain.TimeWindow
		overlap bool
	}{{"touch after", domain.TimeWindow{Start: base.Add(time.Hour), End: base.Add(2 * time.Hour)}, false}, {"inside", domain.TimeWindow{Start: base.Add(15 * time.Minute), End: base.Add(30 * time.Minute)}, true}, {"contains", domain.TimeWindow{Start: base.Add(-time.Hour), End: base.Add(2 * time.Hour)}, true}, {"before", domain.TimeWindow{Start: base.Add(-2 * time.Hour), End: base.Add(-time.Hour)}, false}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := left.Overlaps(tc.other); got != tc.overlap {
				t.Fatalf("got %v", got)
			}
		})
	}
}
func TestRolePermissions(t *testing.T) {
	for _, tc := range []struct {
		role domain.Role
		p    domain.Permission
		want bool
	}{{domain.RoleTeacher, domain.PermissionCreateBatch, true}, {domain.RoleTeacher, domain.PermissionApprove, false}, {domain.RoleStudent, domain.PermissionCheckOut, true}, {domain.RoleStudent, domain.PermissionManageMaintenance, false}, {domain.RoleAdmin, domain.PermissionApprove, true}, {domain.RoleSafety, domain.PermissionReportIncident, true}, {domain.RoleSafety, domain.PermissionCheckOut, false}} {
		t.Run(string(tc.role)+string(tc.p), func(t *testing.T) {
			if got := domain.Allowed(tc.role, tc.p); got != tc.want {
				t.Fatalf("got %v", got)
			}
		})
	}
}
func TestRequestStateMachine(t *testing.T) {
	for _, tc := range []struct {
		from, to domain.RequestStatus
		ok       bool
	}{{domain.RequestPending, domain.RequestApproved, true}, {domain.RequestPending, domain.RequestRejected, true}, {domain.RequestPending, domain.RequestCancelled, true}, {domain.RequestApproved, domain.RequestCancelled, true}, {domain.RequestApproved, domain.RequestRejected, false}, {domain.RequestRejected, domain.RequestApproved, false}} {
		if got := tc.from.CanMove(tc.to); got != tc.ok {
			t.Fatalf("%s -> %s got %v", tc.from, tc.to, got)
		}
	}
}
func TestReservationStateMachine(t *testing.T) {
	valid := [][2]domain.ReservationStatus{{domain.ReservationHeld, domain.ReservationCheckedOut}, {domain.ReservationHeld, domain.ReservationNoShow}, {domain.ReservationHeld, domain.ReservationSuspended}, {domain.ReservationCheckedOut, domain.ReservationReturned}, {domain.ReservationCheckedOut, domain.ReservationSuspended}, {domain.ReservationSuspended, domain.ReservationReturned}}
	for _, pair := range valid {
		if !pair[0].CanMove(pair[1]) {
			t.Fatalf("expected transition")
		}
	}
	for _, pair := range [][2]domain.ReservationStatus{{domain.ReservationReturned, domain.ReservationHeld}, {domain.ReservationNoShow, domain.ReservationReturned}, {domain.ReservationHeld, domain.ReservationReturned}} {
		if pair[0].CanMove(pair[1]) {
			t.Fatalf("unexpected transition")
		}
	}
}
func TestBatchValidation(t *testing.T) {
	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	good := domain.CourseBatch{Title: "CNC", StudentCount: 12, Qualification: "CNC-1", Window: domain.TimeWindow{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour)}}
	if err := domain.ValidateBatch(good, now); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []domain.CourseBatch{{StudentCount: 1, Qualification: "x", Window: good.Window}, {Title: "x", StudentCount: 0, Qualification: "x", Window: good.Window}, {Title: "x", StudentCount: 1, Qualification: "", Window: good.Window}, {Title: "x", StudentCount: 1, Qualification: "x", Window: domain.TimeWindow{Start: now, End: now}}} {
		if err := domain.ValidateBatch(bad, now); err == nil {
			t.Fatalf("expected invalid batch %#v", bad)
		}
	}
}
func TestQuantityAndSeverity(t *testing.T) {
	for _, n := range []int{1, 5, 1000} {
		if err := domain.ValidateQuantity(n); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []int{0, -1, 1001} {
		if err := domain.ValidateQuantity(n); err == nil {
			t.Fatalf("quantity %d accepted", n)
		}
	}
	for _, s := range []string{"low", "medium", "high", "critical"} {
		if err := domain.ValidateSeverity(s); err != nil {
			t.Fatal(err)
		}
	}
	if err := domain.ValidateSeverity("urgent"); err == nil {
		t.Fatal("invalid severity accepted")
	}
}
func TestQualificationAndWindows(t *testing.T) {
	if err := domain.ValidateQualification("lathe", []string{"mill"}); err == nil {
		t.Fatal("missing qualification accepted")
	}
	if err := domain.ValidateQualification("lathe", []string{"lathe"}); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 25, 23, 0, 0, 0, time.UTC)
	w := domain.TimeWindow{Start: base, End: base.Add(2 * time.Hour)}
	if !domain.IsCrossDay(w) {
		t.Fatal("cross day not detected")
	}
	if !domain.InSafetyWindow(base.Add(time.Hour), w) {
		t.Fatal("window should be active")
	}
	if domain.InSafetyWindow(base.Add(3*time.Hour), w) {
		t.Fatal("window should be closed")
	}
}
func TestNormalizeEmailAndRole(t *testing.T) {
	if got := domain.NormalizeEmail(" Teacher@Example.COM "); got != "teacher@example.com" {
		t.Fatal(got)
	}
	for _, role := range []domain.Role{domain.RoleTeacher, domain.RoleStudent, domain.RoleAdmin, domain.RoleSafety} {
		if !role.Valid() {
			t.Fatal(role)
		}
	}
	if domain.Role("unknown").Valid() {
		t.Fatal("unknown role valid")
	}
	if err := domain.RequireRole(domain.RoleTeacher, domain.RoleAdmin); !errors.Is(err, domain.ErrForbidden) {
		t.Fatal(err)
	}
}
