package worker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Job func(context.Context) error
type Queue struct {
	jobs        chan Job
	wg          sync.WaitGroup
	maxAttempts int
	baseDelay   time.Duration
}

func New(size, attempts int) *Queue {
	if size < 1 {
		size = 1
	}
	if attempts < 1 {
		attempts = 1
	}
	return &Queue{jobs: make(chan Job, size), maxAttempts: attempts, baseDelay: 25 * time.Millisecond}
}
func (q *Queue) Start(ctx context.Context) {
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case job, ok := <-q.jobs:
				if !ok {
					return
				}
				q.run(ctx, job)
			}
		}
	}()
}
func (q *Queue) Submit(job Job) error {
	select {
	case q.jobs <- job:
		return nil
	default:
		return fmt.Errorf("worker queue full")
	}
}
func (q *Queue) run(ctx context.Context, job Job) {
	for attempt := 1; attempt <= q.maxAttempts; attempt++ {
		if err := job(ctx); err == nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(q.baseDelay * time.Duration(attempt)):
		}
	}
}
func (q *Queue) Stop() { close(q.jobs); q.wg.Wait() }
