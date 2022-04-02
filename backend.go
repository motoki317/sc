package sc

import (
	"sync"

	"github.com/dboslee/lru"
)

// backend represents a cache backend.
// Backend implementations are expected to be goroutine-safe.
type backend[K comparable, V any] interface {
	Get(key K) (v V, ok bool)
	Set(key K, v V)
	Delete(key K)
}

// Interface guard
var (
	_ backend[string, string] = &mapBackend[string, string]{}
	_ backend[string, string] = lruBackend[string, string]{}
)

type mapBackend[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func (m *mapBackend[K, V]) Get(key K) (v V, ok bool) {
	m.mu.RLock()
	v, ok = m.m[key]
	m.mu.RUnlock()
	return
}

func (m *mapBackend[K, V]) Set(key K, v V) {
	m.mu.Lock()
	m.m[key] = v
	m.mu.Unlock()
}

func (m *mapBackend[K, V]) Delete(key K) {
	m.mu.Lock()
	delete(m.m, key)
	m.mu.Unlock()
}

type lruBackend[K comparable, V any] struct {
	*lru.SyncCache[K, V]
}

func (l lruBackend[K, V]) Get(key K) (v V, ok bool) {
	return l.SyncCache.Get(key)
}

func (l lruBackend[K, V]) Set(key K, v V) {
	l.SyncCache.Set(key, v)
}

func (l lruBackend[K, V]) Delete(key K) {
	l.SyncCache.Delete(key)
}
