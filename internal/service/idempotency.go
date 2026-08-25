package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

type Idempotency struct {
	mu     sync.Mutex
	values map[string]string
}

func NewIdempotency() *Idempotency { return &Idempotency{values: map[string]string{}} }
func (i *Idempotency) Do(ctx context.Context, key string, payload []byte, fn func() error) error {
	if key == "" {
		return fn()
	}
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])
	i.mu.Lock()
	defer i.mu.Unlock()
	if old, ok := i.values[key]; ok {
		if old != digest {
			return fmt.Errorf("idempotency key reused with different payload")
		}
		return nil
	}
	if err := fn(); err != nil {
		return err
	}
	i.values[key] = digest
	return nil
}
