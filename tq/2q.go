package tq

import (
	"github.com/motoki317/sc/lru"
)

// Cache is a fixed size 2Q cache.
// 2Q is an enhancement over the standard LRU cache
// in that it tracks both frequently and recently used
// entries separately. This avoids a burst in access to new
// entries from evicting frequently used entries. It adds some
// additional tracking overhead to the standard LRU cache, and is
// computationally about 2x the cost, and adds some metadata over
// head. The ARCCache is similar, but does not require setting any
// parameters.
type Cache[K comparable, V any] struct {
	size       int
	recentSize int

	recent      *lru.Cache[K, V]
	frequent    *lru.Cache[K, V]
	recentEvict *lru.Cache[K, struct{}]
}

// New creates a new Cache.
func New[K comparable, V any](size int) *Cache[K, V] {
	const (
		recentRatio = 0.5
		ghostRatio  = 0.5
	)

	// Determine the sub-sizes
	recentSize := int(float64(size) * recentRatio)
	evictSize := int(float64(size) * ghostRatio)

	// Allocate the LRUs
	recent := lru.New[K, V](lru.WithCapacity(size))
	frequent := lru.New[K, V](lru.WithCapacity(size))
	recentEvict := lru.New[K, struct{}](lru.WithCapacity(evictSize))

	// Initialize the cache
	return &Cache[K, V]{
		size:        size,
		recentSize:  recentSize,
		recent:      recent,
		frequent:    frequent,
		recentEvict: recentEvict,
	}
}

// Get looks up a key's value from the cache.
func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	// Check if this is a frequent value
	if value, ok = c.frequent.Get(key); ok {
		return
	}

	// If the value is contained in recent, then we
	// promote it to frequent
	if value, ok = c.recent.Peek(key); ok {
		c.recent.Delete(key)
		c.frequent.Set(key, value)
		return
	}

	// No hit
	return
}

// Set adds a value to the cache.
func (c *Cache[K, V]) Set(key K, value V) {
	// Check if the value is frequently used already,
	// and just update the value
	if _, ok := c.frequent.Peek(key); ok {
		c.frequent.Set(key, value)
		return
	}

	// Check if the value is recently used, and promote
	// the value into the frequent list
	if _, ok := c.recent.Peek(key); ok {
		c.recent.Delete(key)
		c.frequent.Set(key, value)
		return
	}

	// If the value was recently evicted, add it to the
	// frequently used list
	if _, ok := c.recentEvict.Peek(key); ok {
		c.ensureSpace(true)
		c.recentEvict.Delete(key)
		c.frequent.Set(key, value)
		return
	}

	// Add to the recently seen list
	c.ensureSpace(false)
	c.recent.Set(key, value)
}

// ensureSpace is used to ensure we have space in the cache
func (c *Cache[K, V]) ensureSpace(recentEvict bool) {
	// If we have space, nothing to do
	recentLen := c.recent.Len()
	freqLen := c.frequent.Len()
	if recentLen+freqLen < c.size {
		return
	}

	// If the recent buffer is larger than
	// the target, evict from there
	if recentLen > 0 && (recentLen > c.recentSize || (recentLen == c.recentSize && !recentEvict)) {
		k, _, _ := c.recent.DeleteOldest()
		c.recentEvict.Set(k, struct{}{})
		return
	}

	// Remove from the frequent list otherwise
	c.frequent.DeleteOldest()
}

// DeleteIf deletes all elements that match the predicate.
func (c *Cache[K, V]) DeleteIf(predicate func(key K, value V) bool) {
	c.frequent.DeleteIf(predicate)
	c.recent.DeleteIf(predicate)
	// does not add to recentEvict, but that is okay for sc's use-case
}

// Delete removes the provided key from the cache.
func (c *Cache[K, V]) Delete(key K) {
	if c.frequent.Delete(key) {
		return
	}
	if c.recent.Delete(key) {
		return
	}
	if c.recentEvict.Delete(key) {
		return
	}
}

// Purge removes all values from the cache.
func (c *Cache[K, V]) Purge() {
	c.frequent.Flush()
	c.recent.Flush()
	c.recentEvict.Flush()
}
