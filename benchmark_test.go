package sc

import (
	"context"
	"testing"
	"time"
)

func BenchmarkCache_Single(b *testing.B) {
	for _, c := range allCaches {
		c := c
		b.Run(c.name, func(b *testing.B) {
			replaceFn := func(ctx context.Context, key string) (string, error) {
				return "value", nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, c.cacheOpts...)
			if err != nil {
				b.Error(err)
			}

			ctx := context.Background()
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				_, _ = cache.Get(ctx, "key")
			}
		})
	}
}

func BenchmarkCache_Parallel_SameKey(b *testing.B) {
	for _, c := range allCaches {
		c := c
		b.Run(c.name, func(b *testing.B) {
			replaceFn := func(ctx context.Context, key string) (string, error) {
				return "value", nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, c.cacheOpts...)
			if err != nil {
				b.Error(err)
			}

			ctx := context.Background()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, _ = cache.Get(ctx, "key")
				}
			})
		})
	}
}

// BenchmarkCache_Parallel_Zipfian benchmarks caches with simulated real world load - zipfian distributed keys
// and replace func that takes 1ms to load.
func BenchmarkCache_Parallel_Zipfian(b *testing.B) {
	const (
		size = 1000
		s    = 1.001
		v    = 100
	)

	for _, c := range evictingCaches {
		c := c
		b.Run(c.name, func(b *testing.B) {
			replaceFn := func(ctx context.Context, key string) (string, error) {
				time.Sleep(1 * time.Millisecond) // simulate some value that takes 1ms to load
				return "value", nil
			}
			cache, err := New[string, string](replaceFn, 100*time.Millisecond, 200*time.Millisecond, append(append([]CacheOption{}, c.cacheOpts...), WithCapacity(size))...)
			if err != nil {
				b.Error(err)
			}

			ctx := context.Background()
			b.RunParallel(func(pb *testing.PB) {
				nextKey := newZipfian(s, v, size*4)
				for pb.Next() {
					_, _ = cache.Get(ctx, nextKey())
				}
			})
		})
	}
}
