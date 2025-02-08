package internal

import (
	"sync"
)

// SyncPool is a generic wrapper around sync.Pool.
type SyncPool[T any] struct {
	p sync.Pool
}

// Get returns a value from the pool.
func (p *SyncPool[T]) Get() *T {
	var existing = p.p.Get()
	if existing != nil {
		return existing.(*T)
	}
	return nil
}

// Put adds x to the pool.
func (p *SyncPool[T]) Put(x *T) {
	p.p.Put(x)
}
