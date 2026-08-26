package domain

import (
	"fmt"
	"strings"
	"time"
)

func ValidateBatch(batch CourseBatch, now time.Time) error {
	if strings.TrimSpace(batch.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if batch.StudentCount <= 0 {
		return fmt.Errorf("student count must be positive")
	}
	if _, err := NewWindow(batch.Window.Start, batch.Window.End); err != nil {
		return err
	}
	if batch.Window.Start.Before(now.Add(-time.Minute)) {
		return fmt.Errorf("start must be current or future")
	}
	if strings.TrimSpace(batch.Qualification) == "" {
		return fmt.Errorf("qualification is required")
	}
	return nil
}
func ValidateQuantity(quantity int) error {
	if quantity <= 0 || quantity > 1000 {
		return fmt.Errorf("quantity out of range")
	}
	return nil
}
func ValidateSeverity(s string) error {
	switch s {
	case "low", "medium", "high", "critical":
		return nil
	default:
		return fmt.Errorf("unsupported severity")
	}
}
