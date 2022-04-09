package sc

import (
	"fmt"
)

// Stats represents cache metrics.
//
// Cache hit ratio can be calculated as:
// 	(cache hit ratio) = (Hits + GraceHits) / (Hits + GraceHits + Misses)
type Stats struct {
	// Hits is the number of fresh cache hits in Cache.Get or Cache.GetFresh.
	Hits uint64
	// GraceHits is the number of stale cache hits in Cache.Get.
	GraceHits uint64
	// Misses is the number of cache misses in Cache.Get or Cache.GetFresh.
	Misses uint64
	// Replacements is the number of times replaceFn is called.
	// Note that this field is incremented after replaceFn finishes to reduce lock time.
	Replacements uint64
}

// String returns formatted string.
func (s Stats) String() string {
	return fmt.Sprintf(
		"Hits: %d, GraceHits: %d, Misses: %d, Replacements: %d, Hit Ratio: %f",
		s.Hits, s.GraceHits, s.Misses, s.Replacements,
		s.HitRatio(),
	)
}

// HitRatio returns the hit ratio.
func (s Stats) HitRatio() float64 {
	total := s.Hits + s.GraceHits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits+s.GraceHits) / float64(total)
}

// Stats returns cache metrics.
// It is useful for monitoring performance and tuning your cache size/type.
func (c *Cache[K, V]) Stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stats
}
