package sc

// CacheOption represents a single cache option.
// See other package-level functions which return CacheOption for more details.
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
	cacheBackend2Q
)

func defaultConfig() cacheConfig {
	return cacheConfig{
		enableStrictCoalescing: false,
		backend:                cacheBackendMap,
		capacity:               0,
	}
}

// WithMapBackend specifies to use the built-in map for storing cache items (the default).
// Note that the default map backend will not evict old cache items. If your key's cardinality is high, consider using
// other backends such as LRU.
// Initial capacity needs to be non-negative.
func WithMapBackend(initialCapacity int) CacheOption {
	return func(c *cacheConfig) {
		c.backend = cacheBackendMap
		c.capacity = initialCapacity
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

// With2QBackend specifies to use 2Q cache for storing cache items.
// Capacity needs to be greater than 0.
func With2QBackend(capacity int) CacheOption {
	return func(c *cacheConfig) {
		c.backend = cacheBackend2Q
		c.capacity = capacity
	}
}

// EnableStrictCoalescing enables strict coalescing check with a slight overhead; the check prevents requests
// coming later in time to be coalesced with already stale response initiated by requests earlier in time.
// This is similar to the behavior of Cache.Forget, but different in that this does not start a new request until
// the current one finishes or Cache.Forget is called.
//
// This is a generalization of so-called 'zero-time-cache', where the original zero-time-cache behavior is
// achievable with zero freshFor/ttl values.
// see also: https://qiita.com/methane/items/27ccaee5b989fb5fca72 (ja)
//
// This is only useful if the freshFor/ttl value is very short (as in the range of a few hundred milliseconds) or
// the request takes a very long time to finish, and you need fresh values for each response.
// Most users should not need this behavior.
func EnableStrictCoalescing() CacheOption {
	return func(c *cacheConfig) {
		c.enableStrictCoalescing = true
	}
}
