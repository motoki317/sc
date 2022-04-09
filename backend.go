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
	// Callers must replace the instance by return value.
	// Implementations may optionally create a new backend using the given cap, if it gives better performance.
	Purge(cap int) backend[K, V]
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

func (m mapBackend[K, V]) Purge(cap int) backend[K, V] {
	return mapBackend[K, V](make(map[K]V, cap))
}

type lruBackend[K comparable, V any] struct {
	*lru.Cache[K, V]
	// As of Go 1.18, Go does not allow type alias of generic types, so we cannot write it like below and have to fall back
	// to embedding in a struct.
	// 	type lruBackend[K comparable, V any] = *lru.Cache[K, V]
}

func newLRUBackend[K comparable, V any](cap int) backend[K, V] {
	return lruBackend[K, V]{lru.New[K, V](lru.WithCapacity(cap))}
}

func (l lruBackend[K, V]) Delete(key K) {
	l.Cache.Delete(key) // Function signature differs a bit
}

func (l lruBackend[K, V]) Purge(_ int) backend[K, V] {
	l.Cache.Flush()
	return l
}

type twoQueueBackend[K comparable, V any] struct {
	*tq.Cache[K, V]
	// Same as lruBackend - cannot use type alias.
}

func new2QBackend[K comparable, V any](cap int) backend[K, V] {
	return twoQueueBackend[K, V]{tq.New[K, V](cap)}
}

func (c twoQueueBackend[K, V]) Purge(_ int) backend[K, V] {
	c.Cache.Purge()
	return c
}
