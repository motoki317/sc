package sc

import (
	"github.com/motoki317/lru"

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

func (m mapBackend[K, V]) Purge() {
	// This form is optimized by the Go-compiler; it calls faster internal mapclear() instead of looping, and avoids
	// allocating new memory.
	// https://go.dev/doc/go1.11#performance
	for key := range m {
		delete(m, key)
	}
}

type lruBackend[K comparable, V any] struct {
	*lru.Cache[K, V]
}

func newLRUBackend[K comparable, V any](cap int) backend[K, V] {
	return lruBackend[K, V]{lru.New[K, V](lru.WithCapacity(cap))}
}

func (l lruBackend[K, V]) Delete(key K) {
	l.Cache.Delete(key) // Function signature differs a bit
}

func (l lruBackend[K, V]) Purge() {
	l.Cache.Flush()
}

type twoQueueBackend[K comparable, V any] struct {
	*tq.Cache[K, V]
	// As of Go 1.18, Go does not allow type alias of generic types, so we cannot write it like below and have to fall back
	// to embedding in a struct.
	// 	type twoQueueBackend[K comparable, V any] = *tq.Cache[K, V]
}

func new2QBackend[K comparable, V any](cap int) backend[K, V] {
	return twoQueueBackend[K, V]{tq.New[K, V](cap)}
}
