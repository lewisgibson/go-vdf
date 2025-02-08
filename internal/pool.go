// Package internal provides internal utilities for the govdf package.
// This package contains implementation details that are not part of the public API.
package internal

import (
	"sync"
)

// Pool is a type-safe Pool of any type T that provides efficient reuse of objects.
// This is a generic wrapper around sync.Pool that provides type safety and eliminates
// the need for type assertions when getting and putting values.
type Pool[T any] struct {
	pool sync.Pool
}

// NewPool creates a new Pool[T] with the provided constructor function.
// The constructor function is called when the pool needs to create a new object
// and no objects are available for reuse.
func NewPool[T any](newFn func() *T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return newFn()
			},
		},
	}
}

// Get returns a value of type T from the pool.
// If no objects are available, a new one is created using the constructor function.
func (p *Pool[T]) Get() *T {
	return p.pool.Get().(*T)
}

// Put returns a value of type T to the pool for reuse.
// The value should be reset to its initial state before being put back into the pool.
func (p *Pool[T]) Put(x *T) {
	p.pool.Put(x)
}
