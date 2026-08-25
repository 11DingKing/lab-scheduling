package clock_test

import (
	"github.com/11DingKing/lab-scheduling/internal/clock"
	"testing"
	"time"
)

func TestFixedClock(t *testing.T) {
	want := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	if got := (clock.Fixed{Value: want}).Now(); !got.Equal(want) {
		t.Fatal(got)
	}
}
func TestRealClockIsRecent(t *testing.T) {
	before := time.Now().UTC()
	got := clock.Real{}.Now()
	after := time.Now().UTC()
	if got.Before(before) || got.After(after) {
		t.Fatalf("got=%v", got)
	}
}
