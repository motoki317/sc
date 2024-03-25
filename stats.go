package sc

import (
	"fmt"
)

type HitStats struct {
	// Hits is the number of fresh cache hits in (*Cache).Get.
	Hits uint64
	// GraceHits is the number of stale cache hits in (*Cache).Get.
	GraceHits uint64
	// Misses is the number of cache misses in (*Cache).Get.
	Misses uint64
	// Replacements is the number of times replaceFn is called.
	// Note that this field is incremented after replaceFn finishes to reduce lock time.
	Replacements uint64
}

type SizeStats struct {
	// Size is the current number of items in the cache.
	Size int
	// Capacity is the maximum number of allowed items in the cache.
	//
	// Note that, for map backend, there is no upper bound in number of items in the cache;
	// Capacity only represents the current cap() of the map.
	Capacity int
}

// Stats represents cache metrics.
type Stats struct {
	HitStats
	SizeStats
}

// String returns formatted string.
func (s Stats) String() string {
	return fmt.Sprintf(
		"Hits: %d, GraceHits: %d, Misses: %d, Replacements: %d, Hit Ratio: %f, Size: %d, Capacity: %d",
		s.Hits, s.GraceHits, s.Misses, s.Replacements,
		s.HitRatio(),
		s.Size, s.Capacity,
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
func (c *cache[K, V]) Stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Stats{
		HitStats: c.stats,
		SizeStats: SizeStats{
			Size:     c.values.Size(),
			Capacity: c.values.Capacity(),
		},
	}
}
