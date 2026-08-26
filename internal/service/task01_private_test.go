package service_test

import (
	"context"
	"github.com/11DingKing/lab-scheduling/internal/service"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentIdempotencyRunsOperationOnce(t *testing.T) {
	idempotency := service.NewIdempotency()
	var calls atomic.Int32
	release := make(chan struct{})
	operation := func() error {
		if calls.Add(1) == 1 {
			select {
			case <-release:
			case <-time.After(100 * time.Millisecond):
			}
		}
		return nil
	}
	done := make(chan error, 2)
	go func() { done <- idempotency.Do(context.Background(), "booking-42", []byte("same"), operation) }()
	time.Sleep(10 * time.Millisecond)
	go func() { done <- idempotency.Do(context.Background(), "booking-42", []byte("same"), operation) }()
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	close(release)
	if got := calls.Load(); got != 1 {
		t.Fatalf("idempotency operation executed %d times", got)
	}
}
