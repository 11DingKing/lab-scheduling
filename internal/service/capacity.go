package service

import (
	"context"
	"fmt"
	"github.com/11DingKing/lab-scheduling/internal/domain"
)

type CapacityStore interface {
	Equipment(context.Context, string) (domain.Equipment, error)
	ActiveReservationCount(context.Context, string) (int, error)
}

func CheckCapacity(ctx context.Context, store CapacityStore, equipmentID string, requested int) error {
	if requested <= 0 {
		return fmt.Errorf("requested capacity must be positive")
	}
	e, err := store.Equipment(ctx, equipmentID)
	if err != nil {
		return err
	}
	if e.Status != domain.EquipmentReady {
		return fmt.Errorf("equipment is not ready: %w", domain.ErrConflict)
	}
	active, err := store.ActiveReservationCount(ctx, equipmentID)
	if err != nil {
		return err
	}
	if active+requested > e.Capacity {
		return domain.ErrCapacity
	}
	return nil
}
func ClampPage(limit, offset int) (int, int) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
