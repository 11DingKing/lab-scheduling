package service_test

import (
	"context"
	"errors"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"github.com/11DingKing/lab-scheduling/internal/service"
	"github.com/11DingKing/lab-scheduling/internal/testutil"
	"testing"
)

func TestCheckCapacity(t *testing.T) {
	db := testutil.Open(t)
	ctx := context.Background()
	if err := service.CheckCapacity(ctx, db, "eq-cnc", 5); err != nil {
		t.Fatal(err)
	}
	if err := service.CheckCapacity(ctx, db, "eq-cnc", 0); err == nil {
		t.Fatal("zero accepted")
	}
	if err := service.CheckCapacity(ctx, db, "eq-cnc", 13); !errors.Is(err, domain.ErrCapacity) {
		t.Fatal(err)
	}
	if err := db.UpdateEquipmentStatus(ctx, "eq-cnc", domain.EquipmentOffline, 1); err != nil {
		t.Fatal(err)
	}
	if err := service.CheckCapacity(ctx, db, "eq-cnc", 1); !errors.Is(err, domain.ErrConflict) {
		t.Fatal(err)
	}
}
func TestClampPage(t *testing.T) {
	for _, tc := range []struct{ limit, offset, wantLimit, wantOffset int }{{0, -1, 20, 0}, {-5, 3, 20, 3}, {200, 2, 100, 2}, {10, 4, 10, 4}} {
		limit, offset := service.ClampPage(tc.limit, tc.offset)
		if limit != tc.wantLimit || offset != tc.wantOffset {
			t.Fatalf("got %d,%d", limit, offset)
		}
	}
}
