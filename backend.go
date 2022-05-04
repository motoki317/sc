package sc

import (
	"github.com/motoki317/sc/lru"
	"github.com/motoki317/sc/tq"
)

// backend represents a cache backend.
// Backend implementations does NOT need to be goroutine-safe.
type backend[K comparable, V any] interface {
	// Get the value for key.
	Get(key K) (v V, ok bool)
	// Set the value for key.
	Set(key K, v V)
	// Delete the value for key.
	Delete(key K)
	// DeleteIf deletes all values that match the predicate.
	DeleteIf(predicate func(key K, value V) bool)
	// Purge all values.
	Purge()
}

type mapBackend[K comparable, V any] map[K]V

func newMapBackend[K comparable, V any](cap int) backend[K, V] {
	return mapBackend[K, V](make(map[K]V, cap))
}

func (m mapBackend[K, V]) Get(key K) (v V, ok bool) {
	v, ok = m[key]
	return
}

func (m mapBackend[K, V]) Set(key K, v V) {
	m[key] = v
}

func (m mapBackend[K, V]) Delete(key K) {
	delete(m, key)
}

func (m mapBackend[K, V]) DeleteIf(predicate func(key K, value V) bool) {
	for k, v := range m {
		if predicate(k, v) {
			delete(m, k)
		}
	}
}

func (m mapBackend[K, V]) Purge() {
	// This form is optimized by the Go-compiler; it calls faster internal mapclear() instead of looping, and avoids
	// allocating new memory.
	// https://go.dev/doc/go1.11#performance
	for key := range m {
		delete(m, key)
	}
}

func newLRUBackend[K comparable, V any](cap int) backend[K, V] {
	return lru.New[K, V](lru.WithCapacity(cap))
}

func new2QBackend[K comparable, V any](cap int) backend[K, V] {
	return tq.New[K, V](cap)
}
