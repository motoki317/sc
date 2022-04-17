package sc

import (
	"context"
	"errors"
	"sync"
	"time"
)

type replaceFunc[K comparable, V any] func(ctx context.Context, key K) (V, error)

// New creates a new cache instance.
// You can specify ttl longer than freshFor to achieve 'graceful cache replacement', where stale item is served via Get
// while a single goroutine is launched in the background to retrieve a fresh item.
func New[K comparable, V any](replaceFn replaceFunc[K, V], freshFor, ttl time.Duration, options ...CacheOption) (*Cache[K, V], error) {
	if replaceFn == nil {
		return nil, errors.New("replaceFn cannot be nil")
	}
	if freshFor < 0 || ttl < 0 {
		return nil, errors.New("freshFor and ttl needs to be non-negative")
	}
	if freshFor > ttl {
		return nil, errors.New("freshFor cannot be longer than ttl")
	}

	config := defaultConfig()
	for _, option := range options {
		option(&config)
	}

	var b backend[K, value[V]]
	switch config.backend {
	case cacheBackendMap:
		if config.capacity < 0 {
			return nil, errors.New("capacity needs to be non-negative for map cache")
		}
		b = newMapBackend[K, value[V]](config.capacity)
	case cacheBackendLRU:
		if config.capacity <= 0 {
			return nil, errors.New("capacity needs to be greater than 0 for LRU cache")
		}
		b = newLRUBackend[K, value[V]](config.capacity)
	case cacheBackend2Q:
		if config.capacity <= 0 {
			return nil, errors.New("capacity needs to be greater than 0 for 2Q cache")
		}
		b = new2QBackend[K, value[V]](config.capacity)
	default:
		return nil, errors.New("unknown cache backend")
	}

	return &Cache[K, V]{
		values:           b,
		calls:            make(map[K]*call[V]),
		fn:               replaceFn,
		freshFor:         freshFor,
		ttl:              ttl,
		strictCoalescing: config.enableStrictCoalescing,
	}, nil
}

// Cache represents a single cache instance.
// All methods are safe to be called from multiple goroutines.
// All cache implementations prevent the 'cache stampede' problem by coalescing multiple requests to the same key.
//
// Notice that Cache doesn't have Set(key K, value V) method - this is intentional. Users are expected to delegate
// the cache replacement logic to Cache by simply calling Get or GetFresh.
type Cache[K comparable, V any] struct {
	values           backend[K, value[V]]
	calls            map[K]*call[V]
	mu               sync.Mutex // mu protects values and calls
	fn               replaceFunc[K, V]
	freshFor, ttl    time.Duration
	strictCoalescing bool
	stats            Stats
}

// Get retrieves an item from the cache.
// Returns the found value and a nil error if found.
// May return a stale item (older than freshFor, but younger than ttl) while a single goroutine is launched
// in the background to update the cache.
// Returns an error as it is if replaceFn returns an error.
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
	return c.get(ctx, key, false)
}

// GetFresh is similar to Get, but if a stale item is found, it waits to retrieve a fresh item instead.
func (c *Cache[K, V]) GetFresh(ctx context.Context, key K) (V, error) {
	return c.get(ctx, key, true)
}

// Forget instructs the cache to forget about the key.
// Corresponding item will be deleted, ongoing cache replacement results (if any) will not be added to the cache,
// and any future Get and GetFresh calls will immediately retrieve a new item.
func (c *Cache[K, V]) Forget(key K) {
	c.mu.Lock()
	if ca, ok := c.calls[key]; ok {
		ca.forgotten = true
	}
	delete(c.calls, key)
	c.values.Delete(key)
	c.mu.Unlock()
}

// Purge instructs the cache to delete all values, and Forget about all ongoing calls.
// Note that frequently calling Purge will worsen the cache performance.
// If you only need to Forget about a specific key, use Forget instead.
func (c *Cache[K, V]) Purge() {
	c.mu.Lock()
	for _, cl := range c.calls {
		cl.forgotten = true
	}
	c.calls = make(map[K]*call[V])
	c.values.Purge()
	c.mu.Unlock()
}

func (c *Cache[K, V]) get(ctx context.Context, key K, needFresh bool) (V, error) {
	// Record time as soon as Get or GetFresh is called *before acquiring the lock* - this maximizes the reuse of values
	t0 := time.Now()
	c.mu.Lock()
	val, ok := c.values.Get(key)

retry:
	// value exists and is fresh - just return
	if ok && val.isFresh(t0, c.freshFor) {
		c.stats.Hits++
		c.mu.Unlock()
		return val.v, nil
	}

	// value exists and is stale, and we're OK with serving it stale while updating in the background
	if ok && !needFresh && !val.isExpired(t0, c.ttl) {
		cl, ok := c.calls[key]
		if !ok {
			cl = &call[V]{}
			cl.wg.Add(1)
			c.calls[key] = cl
			go c.set(ctx, cl, key)
		}
		c.stats.GraceHits++
		c.mu.Unlock()
		return val.v, nil // serve stale contents
	}

	// value doesn't exist or is expired, or is stale, and we need it fresh - sync update
	c.stats.Misses++
	cl, ok := c.calls[key]
	if ok {
		c.mu.Unlock()
		cl.wg.Wait() // make sure not to hold lock while waiting for value
		if c.strictCoalescing && cl.err == nil {
			// Strict request coalescing: compare with the time replaceFn was executed to make sure we are always
			// serving fresh values when needed
			val, ok = cl.val, true // make sure the variables are not shadowed
			c.mu.Lock()            // careful with goto statement - retry is inside critical section
			goto retry
		}
		return cl.val.v, cl.err
	}

	cl = &call[V]{}
	cl.wg.Add(1)
	c.calls[key] = cl
	c.mu.Unlock()

	c.set(ctx, cl, key) // make sure not to hold lock while waiting for value
	return cl.val.v, cl.err
}

func (c *Cache[K, V]) set(ctx context.Context, cl *call[V], key K) {
	// Record time *just before* fn() is called - this maximizes the reuse of values
	cl.val.t = time.Now()
	cl.val.v, cl.err = c.fn(ctx, key)

	c.mu.Lock()
	c.stats.Replacements++
	if !cl.forgotten {
		if cl.err == nil {
			c.values.Set(key, cl.val)
		}
		delete(c.calls, key) // this deletion needs to be inside 'if !cl.forgotten' block, because there may be a new ongoing call
	}
	c.mu.Unlock()
	cl.wg.Done()
}
