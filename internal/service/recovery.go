package service

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Recovery struct {
	Check  func(context.Context) error
	Repair func(context.Context) error
}

func (r Recovery) Run(ctx context.Context) error {
	if r.Check == nil || r.Repair == nil {
		return errors.New("recovery callbacks required")
	}
	if err := r.Check(ctx); err != nil {
		return fmt.Errorf("check recovery: %w", err)
	}
	if err := r.Repair(ctx); err != nil {
		return fmt.Errorf("repair recovery: %w", err)
	}
	return nil
}
func Retry(ctx context.Context, attempts int, delay time.Duration, fn func(context.Context) error) error {
	if attempts < 1 {
		attempts = 1
	}
	var last error
	for i := 0; i < attempts; i++ {
		if err := fn(ctx); err == nil {
			return nil
		} else {
			last = err
		}
		if i+1 < attempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay * time.Duration(i+1)):
			}
		}
	}
	return last
}
