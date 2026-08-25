package worker

import (
	"context"
	"sync"
)

type Registry struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
}

func (r *Registry) Start(parent context.Context, queue *Queue) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.running = true
	queue.Start(ctx)
}
func (r *Registry) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.running = false
}
func (r *Registry) Running() bool { r.mu.Lock(); defer r.mu.Unlock(); return r.running }
