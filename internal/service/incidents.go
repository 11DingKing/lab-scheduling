package service

import (
	"context"
	"fmt"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"github.com/11DingKing/lab-scheduling/internal/repository"
)

type IncidentStore interface {
	CreateIncident(context.Context, repository.IncidentRecord) error
	Incident(context.Context, string) (repository.IncidentRecord, error)
	ResolveIncident(context.Context, string) error
	SuspendReservations(context.Context, string, domain.TimeWindow) error
}

func ReportIncident(ctx context.Context, store IncidentStore, actor domain.User, incident repository.IncidentRecord, equipment domain.Equipment, w domain.TimeWindow, nowUTC string) error {
	if err := domain.RequireRole(actor.Role, domain.RoleSafety, domain.RoleAdmin); err != nil {
		return err
	}
	if err := domain.ValidateSeverity(incident.Severity); err != nil {
		return err
	}
	incident.ReportedBy = actor.ID
	incident.Status = "open"
	if incident.ID == "" {
		return fmt.Errorf("incident id required")
	}
	if err := store.CreateIncident(ctx, incident); err != nil {
		return err
	}
	if err := store.SuspendReservations(ctx, equipment.ID, w); err != nil {
		return err
	}
	return nil
}
func ResolveIncident(ctx context.Context, store IncidentStore, actor domain.User, id string) error {
	if err := domain.RequireRole(actor.Role, domain.RoleSafety, domain.RoleAdmin); err != nil {
		return err
	}
	return store.ResolveIncident(ctx, id)
}
