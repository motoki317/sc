package sc

import (
	"context"
	"errors"
	"sync"
	"time"
)

// replaceFunc is automatically called when value is not present or expired.
// The cache makes sure that replaceFunc is always called once for the same key at the same time.
// When replaceFunc returns an error, value will not be cached.
type replaceFunc[K comparable, V any] func(ctx context.Context, key K) (V, error)

// NewMust is similar to New, but panics on error.
func NewMust[K comparable, V any](replaceFn replaceFunc[K, V], freshFor, ttl time.Duration, options ...CacheOption) *Cache[K, V] {
	c, err := New(replaceFn, freshFor, ttl, options...)
	if err != nil {
		panic(err)
	}
	return c
}

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

	config := defaultConfig(ttl)
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

	c := &Cache[K, V]{
		cache: &cache[K, V]{
			values:           b,
			calls:            make(map[K]*call[V]),
			fn:               replaceFn,
			freshFor:         freshFor,
			ttl:              ttl,
			strictCoalescing: config.enableStrictCoalescing,
		},
	}

	if config.cleanupInterval > 0 {
		startCleaner(c, config.cleanupInterval)
	}

	return c, nil
}

// Cache represents a single cache instance.
// All methods are safe to be called from multiple goroutines.
//
// Notice that Cache doesn't have Set(key K, value V) method - this is intentional. Users are expected to delegate
// the cache replacement logic to Cache by simply calling Get.
type Cache[K comparable, V any] struct {
	*cache[K, V]
	// Embedding must be a pointer to cache, otherwise finalizer is not run.
	// See cleaner doc for the reason Cache and cache is separate.
}

// cache is an internal cache instance.
type cache[K comparable, V any] struct {
	values           backend[K, value[V]]
	calls            map[K]*call[V]
	mu               sync.Mutex // mu protects values and calls
	fn               replaceFunc[K, V]
	freshFor, ttl    time.Duration
	strictCoalescing bool
	stats            Stats
}

// Get retrieves an item. If an item is not in the cache, it automatically loads a new item into the cache.
// May return a stale item (older than freshFor, but younger than ttl) while a new item is being fetched in the background.
// Returns an error as it is if replaceFn returns an error.
//
// The cache prevents 'cache stampede' problem by coalescing multiple requests to the same key.
func (c *cache[K, V]) Get(ctx context.Context, key K) (V, error) {
	// Record time as soon as Get is called *before acquiring the lock* - this maximizes the reuse of values
	calledAt := monoTimeNow()
	c.mu.Lock()
	val, ok := c.values.Get(key)

retry:
	// value exists and is fresh - just return
	if ok && val.isFresh(calledAt, c.freshFor) {
		c.stats.Hits++
		c.mu.Unlock()
		return val.v, nil
	}

	// value exists and is stale - serve it stale while updating in the background
	if ok && !val.isExpired(calledAt, c.ttl) {
		_, ok := c.calls[key]
		if !ok {
			cl := &call[V]{}
			cl.wg.Add(1)
			c.calls[key] = cl
			go c.set(context.Background(), cl, key) // Use empty context so as not to be cancelled by the original context
		}
		c.stats.GraceHits++
		c.mu.Unlock()
		return val.v, nil
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

// GetIfExists retrieves an item without triggering value replacements.
//
// This method doesn't wait for value replacement to finish, even if there is an ongoing one.
func (c *cache[K, V]) GetIfExists(key K) (v V, ok bool) {
	// Record time as soon as Get is called *before acquiring the lock* - this maximizes the reuse of values
	calledAt := monoTimeNow()
	c.mu.Lock()
	defer c.mu.Unlock()
	val, ok := c.values.Get(key)

	// value exists (includes stale values)
	if ok && !val.isExpired(calledAt, c.ttl) {
		if val.isFresh(calledAt, c.freshFor) {
			c.stats.Hits++
		} else {
			c.stats.GraceHits++
		}
		return val.v, true
	}

	// value doesn't exist, or is expired
	c.stats.Misses++
	return val.v, false
}

// Notify instructs the cache to retrieve value for key if value does not exist or is stale, in a non-blocking manner.
func (c *cache[K, V]) Notify(key K) {
	// Record time as soon as Get is called *before acquiring the lock* - this maximizes the reuse of values
	calledAt := monoTimeNow()
	c.mu.Lock()
	val, ok := c.values.Get(key)

	// value exists and is fresh - do nothing
	if ok && val.isFresh(calledAt, c.freshFor) {
		c.mu.Unlock()
		return
	}

	// value exists and is stale, or value doesn't exist - launch goroutine to update in the background
	_, ok = c.calls[key]
	if !ok {
		cl := &call[V]{}
		cl.wg.Add(1)
		c.calls[key] = cl
		go c.set(context.Background(), cl, key) // Use empty context so as not to be cancelled by the original context
	}
	c.mu.Unlock()
}

// Forget instructs the cache to forget about the key.
// Corresponding item will be deleted, ongoing cache replacement results (if any) will not be added to the cache,
// and any future Get calls will immediately retrieve a new item.
func (c *cache[K, V]) Forget(key K) {
	c.mu.Lock()
	delete(c.calls, key)
	c.values.Delete(key)
	c.mu.Unlock()
}

// ForgetIf instructs the cache to Forget about all keys that match the predicate.
func (c *cache[K, V]) ForgetIf(predicate func(key K) bool) {
	c.mu.Lock()
	for key := range c.calls {
		if predicate(key) {
			delete(c.calls, key)
		}
	}
	c.values.DeleteIf(func(key K, _ value[V]) bool { return predicate(key) })
	c.mu.Unlock()
}

// Purge instructs the cache to Forget about all keys.
//
// Note that frequently calling Purge may affect the hit ratio.
// If you only need to Forget about a specific key, use Forget or ForgetIf instead.
func (c *cache[K, V]) Purge() {
	c.mu.Lock()
	for key := range c.calls {
		delete(c.calls, key)
	}
	c.values.Purge()
	c.mu.Unlock()
}

func (c *cache[K, V]) set(ctx context.Context, cl *call[V], key K) {
	// Record time *just before* fn() is called - this maximizes the reuse of values.
	// It is a mistake to set created after fn finishes, otherwise Get may incorrectly return expired values as fresh.
	cl.val.created = monoTimeNow()
	cl.val.v, cl.err = c.fn(ctx, key)

	c.mu.Lock()
	c.stats.Replacements++
	if c.calls[key] == cl {
		if cl.err == nil {
			c.values.Set(key, cl.val)
		}
		delete(c.calls, key) // this deletion needs to be inside 'if c.calls[key] == cl' block, because there may be a new ongoing call
	}
	c.mu.Unlock()
	cl.wg.Done()
}

// cleanup cleans up expired items from the cache, freeing memory.
func (c *cache[K, V]) cleanup() {
	c.mu.Lock()
	now := monoTimeNow() // Record time after acquiring the lock to maximize freeing of expired items
	c.values.DeleteIf(func(key K, value value[V]) bool {
		return value.isExpired(now, c.ttl)
	})
	c.mu.Unlock()
}
