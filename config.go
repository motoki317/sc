package sc

type CacheOption func(c *cacheConfig)

type cacheConfig struct {
	enableStrictCoalescing bool
	backend                cacheBackendType
	capacity               int
}

type cacheBackendType int

const (
	cacheBackendMap cacheBackendType = iota
	cacheBackendLRU
	cacheBackendARC
)

func defaultConfig() cacheConfig {
	return cacheConfig{
		enableStrictCoalescing: false,
		backend:                cacheBackendMap,
		capacity:               0,
	}
}

// WithCapacity sets the cache's capacity.
func WithCapacity(capacity int) CacheOption {
	return func(c *cacheConfig) {
		c.capacity = capacity
	}
}

// WithMapBackend specifies to use the built-in map for storing cache items (the default).
// Note that the default map backend will not evict old cache items. If your key's cardinality is high, consider using
// other backends such as LRU.
func WithMapBackend() CacheOption {
	return func(c *cacheConfig) {
		c.backend = cacheBackendMap
	}
}

// WithLRUBackend specifies to use LRU for storing cache items.
// Capacity needs to be greater than 0.
func WithLRUBackend(capacity int) CacheOption {
	return func(c *cacheConfig) {
		c.backend = cacheBackendLRU
		c.capacity = capacity
	}
}

// WithARCBackend specifies to use ARC for storing cache items.
// Capacity needs to be greater than 0.
func WithARCBackend(capacity int) CacheOption {
	return func(c *cacheConfig) {
		c.backend = cacheBackendARC
		c.capacity = capacity
	}
}

// EnableStrictCoalescing enables strict coalescing check with a slight overhead; the check prevents requests
// coming later in time to be coalesced with already stale response initiated by requests earlier in time.
// This is similar to the behavior of Cache.Forget, but different in that this does not start a new request until
// the current one finishes or Cache.Forget is called.
//
// This is a generalization of so-called 'zero-time-cache', where the original zero-time-cache behavior is
// achievable with zero freshFor/ttl values. cf: https://qiita.com/methane/items/27ccaee5b989fb5fca72
//
// This is only useful if the freshFor/ttl value is very short (as in the range of a few hundred milliseconds) or
// the request takes a very long time to finish, and you need fresh values for each response.
// Most users should not need this behavior.
func EnableStrictCoalescing() CacheOption {
	return func(c *cacheConfig) {
		c.enableStrictCoalescing = true
	}
}
