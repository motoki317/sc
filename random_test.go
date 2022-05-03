package sc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCache_GetRandom ensures that the cache returns correct values.
func TestCache_GetRandom(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			replaceFn := func(ctx context.Context, key string) (string, error) {
				time.Sleep(1 * time.Millisecond)
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 50*time.Millisecond, 50*time.Millisecond, c.cacheOpts...)
			assert.NoError(t, err)

			keys := newKeys(newZipfian(1.001, 50, 100), 1000)
			for _, key := range keys {
				val, err := cache.Get(context.Background(), key)
				assert.NoError(t, err)
				assert.Equal(t, "result-"+key, val)
			}
		})
	}
}

// TestCache_GetRandom_GracefulReplacement is similar to TestCache_GetRandom, but uses graceful cache replacement.
func TestCache_GetRandom_GracefulReplacement(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			replaceFn := func(ctx context.Context, key string) (string, error) {
				time.Sleep(1 * time.Millisecond)
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 10*time.Millisecond, 20*time.Millisecond, c.cacheOpts...)
			assert.NoError(t, err)

			keys := newKeys(newZipfian(1.001, 50, 100), 1000)
			for _, key := range keys {
				val, err := cache.Get(context.Background(), key)
				assert.NoError(t, err)
				assert.Equal(t, "result-"+key, val)
			}
		})
	}
}

// TestCache_GetRandom_Parallel is similar to TestCache_GetRandom, but in parallel.
func TestCache_GetRandom_Parallel(t *testing.T) {
	t.Parallel()

	const (
		concurrency = 25
		cacheSize   = 100
		s           = 1.01
		v           = 10
	)
	for _, c := range allCaches(cacheSize) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			replaceFn := func(ctx context.Context, key string) (string, error) {
				time.Sleep(1 * time.Millisecond) // sleep for some time to simulate concurrent access
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 10*time.Millisecond, 10*time.Millisecond, c.cacheOpts...)
			assert.NoError(t, err)

			var wg sync.WaitGroup
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					keys := newKeys(newZipfian(s, v, cacheSize*2), cacheSize*10)
					for _, key := range keys {
						val, err := cache.Get(context.Background(), key)
						assert.NoError(t, err)
						assert.Equal(t, "result-"+key, val)
					}
				}()
			}
			wg.Wait()
		})
	}
}

// TestCache_GetRandom_Parallel_GracefulReplacement is similar to TestCache_GetRandom_Parallel, but uses graceful cache replacement.
func TestCache_GetRandom_Parallel_GracefulReplacement(t *testing.T) {
	t.Parallel()

	const (
		concurrency = 25
		cacheSize   = 100
		s           = 1.01
		v           = 10
	)
	for _, c := range allCaches(cacheSize) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			replaceFn := func(ctx context.Context, key string) (string, error) {
				time.Sleep(1 * time.Millisecond) // sleep for some time to simulate concurrent access
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 10*time.Millisecond, 20*time.Millisecond, c.cacheOpts...)
			assert.NoError(t, err)

			var wg sync.WaitGroup
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					keys := newKeys(newZipfian(s, v, cacheSize*2), cacheSize*10)
					for _, key := range keys {
						val, err := cache.Get(context.Background(), key)
						assert.NoError(t, err)
						assert.Equal(t, "result-"+key, val)
					}
				}()
			}
			wg.Wait()
		})
	}
}
