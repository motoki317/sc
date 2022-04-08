package sc

import (
	"github.com/motoki317/lru"

	"github.com/motoki317/sc/tq"
)

// backend represents a cache backend.
// Backend implementations does NOT need to be goroutine-safe.
type backend[K comparable, V any] interface {
	Get(key K) (v V, ok bool)
	Set(key K, v V)
	Delete(key K)
}

// Interface guard
var (
	_ backend[string, string] = mapBackend[string, string]{}
	_ backend[string, string] = lruBackend[string, string]{}
	_ backend[string, string] = twoQueueBackend[string, string]{}
)

type mapBackend[K comparable, V any] map[K]V

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

type lruBackend[K comparable, V any] struct {
	c *lru.Cache[K, V]
}

func (l lruBackend[K, V]) Get(key K) (v V, ok bool) {
	return l.c.Get(key)
}

func (l lruBackend[K, V]) Set(key K, v V) {
	l.c.Set(key, v)
}

func (l lruBackend[K, V]) Delete(key K) {
	l.c.Delete(key) // The signature differs by a bit (lru.Cache returns bool) so we cannot use embedding
}

// twoQueueBackend represents 2Q cache backend.
//
// As of Go 1.18, Go does not allow type alias of generic types, so we cannot write it like below and have to fall back
// to embedding in a struct.
// 	type twoQueueBackend[K comparable, V any] = *tq.Cache[K, V]
type twoQueueBackend[K comparable, V any] struct {
	*tq.Cache[K, V]
}
