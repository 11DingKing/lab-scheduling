package worker

import (
	"context"
	"sync"
	"time"
)

type DeadLetter struct {
	Job       Job
	Attempts  int
	LastError string
	FailedAt  time.Time
}
type DeadLetters struct {
	mu    sync.Mutex
	items []DeadLetter
}

func (d *DeadLetters) Add(item DeadLetter) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.items = append(d.items, item)
}
func (d *DeadLetters) List() []DeadLetter {
	d.mu.Lock()
	defer d.mu.Unlock()
	copyItems := make([]DeadLetter, len(d.items))
	copy(copyItems, d.items)
	return copyItems
}
func RunOnce(ctx context.Context, job Job, dead *DeadLetters) error {
	if err := job(ctx); err != nil {
		if dead != nil {
			dead.Add(DeadLetter{Job: job, Attempts: 1, LastError: err.Error(), FailedAt: time.Now().UTC()})
		}
		return err
	}
	return nil
}
