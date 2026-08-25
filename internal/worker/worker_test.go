package worker_test

import (
	"context"
	"errors"
	"github.com/11DingKing/lab-scheduling/internal/worker"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueueRetriesAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q := worker.New(4, 3)
	q.Start(ctx)
	var calls atomic.Int32
	done := make(chan struct{})
	if err := q.Submit(func(context.Context) error {
		n := calls.Add(1)
		if n < 3 {
			return errors.New("retry")
		}
		close(done)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
	if calls.Load() != 3 {
		t.Fatal(calls.Load())
	}
	q.Stop()
}
func TestQueueRejectsFull(t *testing.T) {
	q := worker.New(1, 1)
	block := make(chan struct{})
	if err := q.Submit(func(context.Context) error { <-block; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := q.Submit(func(context.Context) error { return nil }); err == nil {
		t.Fatal("full queue accepted")
	}
	close(block)
}
func TestQueueCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	q := worker.New(1, 3)
	q.Start(ctx)
	started := make(chan struct{})
	finished := make(chan struct{})
	if err := q.Submit(func(ctx context.Context) error { close(started); <-ctx.Done(); close(finished); return ctx.Err() }); err != nil {
		t.Fatal(err)
	}
	<-started
	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("cancel not propagated")
	}
	q.Stop()
}
func TestRegistryLifecycle(t *testing.T) {
	q := worker.New(2, 1)
	var r worker.Registry
	if r.Running() {
		t.Fatal("running before start")
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.Start(ctx, q)
	if !r.Running() {
		t.Fatal("not running")
	}
	r.Start(ctx, q)
	r.Stop()
	if r.Running() {
		t.Fatal("running after stop")
	}
	cancel()
}
