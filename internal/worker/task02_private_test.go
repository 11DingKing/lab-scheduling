package worker_test

import (
	"context"
	"testing"
	"time"
	"github.com/11DingKing/lab-scheduling/internal/worker"
)

func TestWorkerPropagatesCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	queue := worker.New(2, 1)
	registry := &worker.Registry{}
	registry.Start(parent, queue)
	started := make(chan struct{})
	finished := make(chan struct{})
	if err := queue.Submit(func(ctx context.Context) error { close(started); <-ctx.Done(); close(finished); return ctx.Err() }); err != nil { t.Fatal(err) }
	select { case <-started: case <-time.After(time.Second): t.Fatal("job did not start") }
	cancel()
	select { case <-finished: case <-time.After(time.Second): t.Fatal("worker ignored parent cancellation") }
	registry.Stop()
}
