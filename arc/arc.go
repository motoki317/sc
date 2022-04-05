package arc

import (
	"sync"

	"github.com/motoki317/lru"
)

// Below includes modified code from https://github.com/hashicorp/golang-lru/blob/80c98217689d6df152309d574ccc682b21dc802c/arc.go.

// Cache is a thread-safe fixed size Adaptive Replacement Cache (ARC).
// ARC is an enhancement over the standard LRU cache in that tracks both
// frequency and recency of use. This avoids a burst in access to new
// entries from evicting the frequently used older entries. It adds some
// additional tracking overhead to a standard LRU cache, computationally
// it is roughly 2x the cost, and the extra memory overhead is linear
// with the size of the cache. ARC has been patented by IBM, but is
// similar to the TwoQueueCache (2Q) which requires setting parameters.
type Cache[K comparable, V any] struct {
	size int // Size is the total capacity of the cache
	p    int // p is the dynamic preference towards t1 over t2

	t1 *lru.Cache[K, V]        // t1 is the LRU for recently accessed items
	b1 *lru.Cache[K, struct{}] // b1 is the LRU for evictions from t1

	t2 *lru.Cache[K, V]        // t2 is the LRU for frequently accessed items
	b2 *lru.Cache[K, struct{}] // b2 is the LRU for evictions from t2

	lock sync.RWMutex
}

// New creates an ARC of the given size.
func New[K comparable, V any](size int) *Cache[K, V] {
	// Create the sub LRUs
	b1 := lru.New[K, struct{}](lru.WithCapacity(size))
	b2 := lru.New[K, struct{}](lru.WithCapacity(size))
	t1 := lru.New[K, V](lru.WithCapacity(size))
	t2 := lru.New[K, V](lru.WithCapacity(size))

	// Initialize the ARC
	return &Cache[K, V]{
		size: size,
		p:    size / 2,
		t1:   t1,
		b1:   b1,
		t2:   t2,
		b2:   b2,
	}
}

// Get looks up a key's value from the cache.
func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// If the value is contained in T1 (recent), then
	// promote it to T2 (frequent)
	if val, ok := c.t1.Peek(key); ok {
		c.t1.Delete(key)
		c.t2.Set(key, val)
		return val, ok
	}

	// Check if the value is contained in T2 (frequent)
	if val, ok := c.t2.Get(key); ok {
		return val, ok
	}

	// No hit
	return
}

// Set adds a value to the cache.
func (c *Cache[K, V]) Set(key K, value V) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Check if the value is contained in T1 (recent), and potentially
	// promote it to frequent T2
	if _, ok := c.t1.Peek(key); ok {
		c.t1.Delete(key)
		c.t2.Set(key, value)
		return
	}

	// Check if the value is already in T2 (frequent) and update it
	if _, ok := c.t2.Peek(key); ok {
		c.t2.Set(key, value)
		return
	}

	// Check if this value was recently evicted as part of the
	// recently used list
	if _, ok := c.b1.Peek(key); ok {
		// T1 set is too small, increase P appropriately
		delta := 1
		b1Len := c.b1.Len()
		b2Len := c.b2.Len()
		if b2Len > b1Len {
			delta = b2Len / b1Len
		}
		if c.p+delta >= c.size {
			c.p = c.size
		} else {
			c.p += delta
		}

		// Remove from B1
		c.b1.Delete(key)

		// Add the key to the frequently used list
		c.t2.Set(key, value)

		// Potentially need to make room in the cache
		c.replace()
		return
	}

	// Check if this value was recently evicted as part of the
	// frequently used list
	if _, ok := c.b2.Peek(key); ok {
		// T2 set is too small, decrease P appropriately
		delta := 1
		b1Len := c.b1.Len()
		b2Len := c.b2.Len()
		if b1Len > b2Len {
			delta = b1Len / b2Len
		}
		if delta >= c.p {
			c.p = 0
		} else {
			c.p -= delta
		}

		// Remove from B2
		c.b2.Delete(key)

		// Add the key to the frequently used list
		c.t2.Set(key, value)

		// Potentially need to make room in the cache
		c.replace()
		return
	}

	// Add to the recently seen list
	c.t1.Set(key, value)

	// Potentially need to make room in the cache
	c.replace()
}

// replace is used to adaptively evict from either T1 or T2
// based on the current learned value of P
func (c *Cache[K, V]) replace() {
	if c.t1.Len()+c.t2.Len() <= c.size {
		return
	}
	if c.t1.Len() > c.p {
		k, _, ok := c.t1.DeleteOldest()
		if ok {
			c.b1.Set(k, struct{}{})
			if c.b1.Len() > c.size-c.p {
				c.b1.DeleteOldest()
			}
		}
	} else {
		k, _, ok := c.t2.DeleteOldest()
		if ok {
			c.b2.Set(k, struct{}{})
			if c.b2.Len() > c.p {
				c.b2.DeleteOldest()
			}
		}
	}
}

// Delete is used to purge a key from the cache
func (c *Cache[K, V]) Delete(key K) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.t1.Delete(key) {
		return
	}
	if c.t2.Delete(key) {
		return
	}
	if c.b1.Delete(key) {
		return
	}
	if c.b2.Delete(key) {
		return
	}
}
