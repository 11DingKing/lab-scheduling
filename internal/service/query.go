package service

import (
	"context"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"github.com/11DingKing/lab-scheduling/internal/repository"
)

type QueryStore interface {
	ListEquipment(context.Context, string) ([]domain.Equipment, error)
	Audits(context.Context, string, string) ([]repository.AuditEvent, error)
}

func ListReadyEquipment(ctx context.Context, store QueryStore) ([]domain.Equipment, error) {
	return store.ListEquipment(ctx, string(domain.EquipmentReady))
}
func AuditTrail(ctx context.Context, store QueryStore, entity, id string) ([]repository.AuditEvent, error) {
	return store.Audits(ctx, entity, id)
}
