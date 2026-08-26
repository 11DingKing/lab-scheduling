package domain_test

import (
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"testing"
	"time"
)

func TestWindowSerialization(t *testing.T) {
	start := time.Date(2026, 8, 25, 8, 30, 0, 123, time.UTC)
	w := domain.TimeWindow{Start: start, End: start.Add(90 * time.Minute)}
	dto := domain.WindowToDTO(w)
	round, err := domain.WindowFromDTO(dto)
	if err != nil {
		t.Fatal(err)
	}
	if !round.Start.Equal(start) || !round.End.Equal(w.End) {
		t.Fatalf("round=%+v", round)
	}
	if _, err := domain.WindowFromDTO(domain.WindowDTO{Start: "bad", End: dto.End}); err == nil {
		t.Fatal("bad start accepted")
	}
	if _, err := domain.WindowFromDTO(domain.WindowDTO{Start: dto.Start, End: "bad"}); err == nil {
		t.Fatal("bad end accepted")
	}
}
func TestReservationStatusHelpers(t *testing.T) {
	for _, s := range []domain.ReservationStatus{domain.ReservationHeld, domain.ReservationCheckedOut, domain.ReservationSuspended} {
		if !domain.StatusIsActive(s) {
			t.Fatal(s)
		}
		if domain.StatusIsTerminal(s) {
			t.Fatal(s)
		}
	}
	for _, s := range []domain.ReservationStatus{domain.ReservationReturned, domain.ReservationCancelled, domain.ReservationNoShow} {
		if !domain.StatusIsTerminal(s) {
			t.Fatal(s)
		}
		if domain.StatusIsActive(s) {
			t.Fatal(s)
		}
	}
}
func TestWindowDTOEmpty(t *testing.T) {
	if _, err := domain.WindowFromDTO(domain.WindowDTO{}); err == nil {
		t.Fatal("empty window accepted")
	}
}
