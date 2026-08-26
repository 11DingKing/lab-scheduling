package observability_test

import (
	"github.com/11DingKing/lab-scheduling/internal/observability"
	"testing"
)

func TestLoggerEvents(t *testing.T) {
	l := observability.NewLogger()
	l.Info("started", map[string]any{"component": "test"})
	l.Error("failed", map[string]any{"retry": 1})
	l.Event("debug", "details", nil)
}
