package sc

import (
	"context"
	"testing"
	"time"
)

func BenchmarkCache_Single_SameKey(b *testing.B) {
	for _, c := range allCaches(10) {
		c := c
		b.Run(c.name, func(b *testing.B) {
			replaceFn := func(ctx context.Context, key string) (string, error) {
				return "value", nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Minute, 1*time.Minute, c.cacheOpts...)
			if err != nil {
				b.Error(err)
			}

			ctx := context.Background()
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				_, _ = cache.Get(ctx, "key")
			}
			b.Log(cache.Stats())
		})
	}
}

func BenchmarkCache_Single_Zipfian(b *testing.B) {
	const (
		size = 1000
		s    = 1.001
		v    = 100
	)

	for _, c := range allCaches(size) {
		c := c
		b.Run(c.name, func(b *testing.B) {
			replaceFn := func(ctx context.Context, key string) (string, error) {
				return "value", nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Minute, 1*time.Minute, c.cacheOpts...)
			if err != nil {
				b.Error(err)
			}

			ctx := context.Background()
			keys := newKeys(newZipfian(s, v, size*4), size*10)
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				_, _ = cache.Get(ctx, keys[i%(size*10)])
			}
			b.Log(cache.Stats())
		})
	}
}

func BenchmarkCache_Parallel_SameKey(b *testing.B) {
	for _, c := range allCaches(10) {
		c := c
		b.Run(c.name, func(b *testing.B) {
			replaceFn := func(ctx context.Context, key string) (string, error) {
				return "value", nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Minute, 1*time.Minute, c.cacheOpts...)
			if err != nil {
				b.Error(err)
			}

			ctx := context.Background()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, _ = cache.Get(ctx, "key")
				}
			})
			b.Log(cache.Stats())
		})
	}
}

func BenchmarkCache_Parallel_Zipfian(b *testing.B) {
	const (
		size = 1000
		s    = 1.001
		v    = 100
	)

	for _, c := range allCaches(size) {
		c := c
		b.Run(c.name, func(b *testing.B) {
			replaceFn := func(ctx context.Context, key string) (string, error) {
				return "value", nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Minute, 1*time.Minute, c.cacheOpts...)
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
			b.Log(cache.Stats())
		})
	}
}

// BenchmarkCache_RealWorkLoad benchmarks caches with simulated real world load - zipfian distributed keys
// and replace func that takes 1ms to load.
func BenchmarkCache_RealWorkLoad(b *testing.B) {
	const (
		size = 1000
		s    = 1.001
		v    = 100
	)

	// Only benchmark against evicting caches (not the built-in map backend) because the map backend can cache all values.
	for _, c := range evictingCaches(size) {
		c := c
		b.Run(c.name, func(b *testing.B) {
			replaceFn := func(ctx context.Context, key string) (string, error) {
				time.Sleep(1 * time.Millisecond) // simulate some value that takes 1ms to load
				return "value", nil
			}
			cache, err := New[string, string](replaceFn, 100*time.Millisecond, 200*time.Millisecond, c.cacheOpts...)
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
			b.Log(cache.Stats())
		})
	}
}
