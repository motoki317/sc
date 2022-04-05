package sc

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	fn := func(ctx context.Context, s string) (string, error) { return "", nil }

	t.Run("defaults to map", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0)
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, &mapBackend[string, value[string]]{}, c.values)
		assert.False(t, c.strictCoalescing)
	})

	t.Run("invalid backend", func(t *testing.T) {
		t.Parallel()

		// A test case just to increase coverage to 100%
		// Normal users should not be able to reach the "unknown cache backend" path
		_, err := New[string, string](fn, 0, 0, func(c *cacheConfig) { c.backend = -1 })
		assert.Error(t, err)
	})

	t.Run("invalid replaceFn", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](nil, 0, 0)
		assert.Error(t, err)
	})

	t.Run("invalid freshFor", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, -1, 0)
		assert.Error(t, err)
	})

	t.Run("invalid ttl", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 0, -1)
		assert.Error(t, err)
	})

	t.Run("invalid freshFor and ttl configuration", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 2*time.Minute, 1*time.Minute)
		assert.Error(t, err)
	})

	t.Run("map cache", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, WithMapBackend())
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, &mapBackend[string, value[string]]{}, c.values)
		assert.False(t, c.strictCoalescing)
	})

	t.Run("map cache with invalid capacity", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 0, 0, WithMapBackend(), WithCapacity(-1))
		assert.Error(t, err)
	})

	t.Run("map cache with capacity", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, WithMapBackend(), WithCapacity(10))
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, &mapBackend[string, value[string]]{}, c.values)
		assert.False(t, c.strictCoalescing)
	})

	t.Run("strict map cache", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, WithMapBackend(), EnableStrictCoalescing())
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, &mapBackend[string, value[string]]{}, c.values)
		assert.True(t, c.strictCoalescing)
	})

	t.Run("strict map cache with capacity", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, WithMapBackend(), EnableStrictCoalescing(), WithCapacity(10))
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, &mapBackend[string, value[string]]{}, c.values)
		assert.True(t, c.strictCoalescing)
	})

	t.Run("LRU needs capacity set", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 0, 0, WithLRUBackend(10), WithCapacity(0))
		assert.Error(t, err)
	})

	t.Run("LRU cache with invalid capacity", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 0, 0, WithLRUBackend(10), WithCapacity(-1))
		assert.Error(t, err)
	})

	t.Run("struct LRU needs capacity set", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 0, 0, WithLRUBackend(10), EnableStrictCoalescing(), WithCapacity(0))
		assert.Error(t, err)
	})

	t.Run("LRU cache", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, WithLRUBackend(10))
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, lruBackend[string, value[string]]{}, c.values)
		assert.False(t, c.strictCoalescing)
	})

	t.Run("strict LRU cache", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, WithLRUBackend(10), EnableStrictCoalescing())
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, lruBackend[string, value[string]]{}, c.values)
		assert.True(t, c.strictCoalescing)
	})
}

func TestCache_GetRandom(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			replaceFn := func(ctx context.Context, key string) (string, error) {
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, c.cacheOpts...)
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

func TestCache_GetRandom_Parallel(t *testing.T) {
	t.Parallel()

	const (
		concurrency = 25
		cacheSize   = 100
		s           = 1.01
		v           = 10
	)
	for _, c := range allCaches {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			replaceFn := func(ctx context.Context, key string) (string, error) {
				time.Sleep(1 * time.Millisecond) // sleep for some time to simulate concurrent access
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 10*time.Millisecond, 10*time.Millisecond, append(append([]CacheOption{}, c.cacheOpts...), WithCapacity(cacheSize))...)
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

func TestCache_GetRandom_Parallel_GracefulReplacement(t *testing.T) {
	t.Parallel()

	const (
		concurrency = 25
		cacheSize   = 100
		s           = 1.01
		v           = 10
	)
	for _, c := range allCaches {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			replaceFn := func(ctx context.Context, key string) (string, error) {
				time.Sleep(1 * time.Millisecond) // sleep for some time to simulate concurrent access
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 10*time.Millisecond, 20*time.Millisecond, append(append([]CacheOption{}, c.cacheOpts...), WithCapacity(cacheSize))...)
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

func TestCache_Get(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var cnt int64
			replaceFn := func(ctx context.Context, key string) (string, error) {
				t.Log("replaceFn triggered")
				assert.Equal(t, "k1", key)
				atomic.AddInt64(&cnt, 1)
				time.Sleep(500 * time.Millisecond)
				return "result1", nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, c.cacheOpts...)
			assert.NoError(t, err)

			t0 := time.Now()
			// repeat test multiple times
			for x := 0; x < 5; x++ {
				var wg sync.WaitGroup
				for i := 0; i < 10; i++ {
					// watch for number of goroutines to make sure only one goroutine is launched to trigger replaceFn
					// t.Logf("numGoroutines = %d", runtime.NumGoroutine())
					wg.Add(1)
					go func() {
						defer wg.Done()
						val, err := cache.Get(context.Background(), "k1")
						assert.NoError(t, err)
						assert.Equal(t, "result1", val)
					}()
				}
				wg.Wait()

				// ensure single call
				assert.EqualValues(t, 1, cnt)
				// assert t=500ms
				assert.InDelta(t, 500*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}
		})
	}
}

func TestCache_GetError(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			targetErr := errors.New("test error")
			replaceFn := func(ctx context.Context, key string) (string, error) {
				assert.Equal(t, "k1", key)
				return "", targetErr
			}
			cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, c.cacheOpts...)
			assert.NoError(t, err)

			val, err := cache.Get(context.Background(), "k1")
			assert.Zero(t, val)
			assert.Error(t, err)
			assert.Equal(t, targetErr, err)
		})
	}
}

func TestCache_GetFresh(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var cnt int64
			replaceFn := func(ctx context.Context, key string) (string, error) {
				assert.Equal(t, "k1", key)
				// some expensive op..
				time.Sleep(500 * time.Millisecond)
				atomic.AddInt64(&cnt, 1)
				return "result1", nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, c.cacheOpts...)
			assert.NoError(t, err)

			t0 := time.Now()
			// repeat test multiple times
			for x := 0; x < 5; x++ {
				var wg sync.WaitGroup
				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						val, err := cache.GetFresh(context.Background(), "k1")
						assert.NoError(t, err)
						assert.Equal(t, "result1", val)
					}()
				}
				wg.Wait()

				// ensure single call
				assert.EqualValues(t, 1, cnt)
				// assert t=500ms
				assert.InDelta(t, 500*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}
		})
	}
}

func TestCache_GetFresh_Sync(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var cnt int64
			replaceFn := func(ctx context.Context, key string) (string, error) {
				assert.Equal(t, "k1", key)
				atomic.AddInt64(&cnt, 1)
				time.Sleep(500 * time.Millisecond)
				return "result1", nil
			}
			cache, err := New[string, string](replaceFn, 250*time.Millisecond, 1*time.Second, c.cacheOpts...)
			assert.NoError(t, err)

			t0 := time.Now()
			// t=0ms, 1st call group
			var wg sync.WaitGroup
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					val, err := cache.GetFresh(context.Background(), "k1")
					assert.NoError(t, err)
					assert.Equal(t, "result1", val)
				}()
			}
			wg.Wait()
			assert.EqualValues(t, 1, cnt)
			// assert t=500ms
			assert.InDelta(t, 500*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))

			// t=500ms, 2nd call group -> has stale values, but needs to fetch fresh values
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					val, err := cache.GetFresh(context.Background(), "k1")
					assert.NoError(t, err)
					assert.Equal(t, "result1", val)
				}()
			}
			wg.Wait()
			assert.EqualValues(t, 2, cnt)
			// assert t=1000ms
			assert.InDelta(t, 1000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}

func TestCache_Forget_Interrupt(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var cnt int64
			replaceFn := func(ctx context.Context, key string) (string, error) {
				assert.Equal(t, "k1", key)
				atomic.AddInt64(&cnt, 1)
				time.Sleep(750 * time.Millisecond)
				return "result1", nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, c.cacheOpts...)
			assert.NoError(t, err)

			t0 := time.Now()
			var wg sync.WaitGroup
			// t=0ms, 1st call
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "result1", v)
			}()
			time.Sleep(500 * time.Millisecond)
			// t=500ms, Forget, then 2nd call
			cache.Forget("k1")
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "result1", v)
			}()
			wg.Wait()
			// t=1250ms, assert replaceFn was triggered twice
			assert.EqualValues(t, 2, cnt)
			assert.InDelta(t, 1250*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}

func TestCache_Forget_NoInterrupt(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var cnt int64
			replaceFn := func(ctx context.Context, key string) (string, error) {
				assert.Equal(t, "k1", key)
				atomic.AddInt64(&cnt, 1)
				time.Sleep(250 * time.Millisecond)
				return "result1", nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, c.cacheOpts...)
			assert.NoError(t, err)

			t0 := time.Now()
			var wg sync.WaitGroup
			// t=0ms, 1st call
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "result1", v)
			}()
			time.Sleep(500 * time.Millisecond)
			// t=500ms, Forget, then 2nd call
			cache.Forget("k1")
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "result1", v)
			}()
			wg.Wait()
			// t=750ms, assert replaceFn was triggered twice
			assert.EqualValues(t, 2, cnt)
			assert.InDelta(t, 750*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}

func TestCache_MultipleValues(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var cnt int64
			replaceFn := func(ctx context.Context, key string) (string, error) {
				atomic.AddInt64(&cnt, 1)
				time.Sleep(500 * time.Millisecond)
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, c.cacheOpts...)
			assert.NoError(t, err)

			t0 := time.Now()
			var wg sync.WaitGroup
			// t=0ms, 1st group call
			for i := 0; i < 50; i++ {
				k := "k" + strconv.Itoa(i%5)
				wg.Add(1)
				go func() {
					defer wg.Done()
					v, err := cache.Get(context.Background(), k)
					assert.NoError(t, err)
					assert.Equal(t, "result-"+k, v)
					// assert t=500ms
					assert.InDelta(t, 500*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
				}()
			}
			wg.Wait()
			// assert replaceFn was triggered exactly 5 times
			assert.EqualValues(t, 5, cnt)
			// assert t=500ms
			assert.InDelta(t, 500*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))

			time.Sleep(1 * time.Second)
			// t=1500ms, 2nd group call
			for i := 0; i < 50; i++ {
				k := "k" + strconv.Itoa(i%6)
				wg.Add(1)
				go func() {
					defer wg.Done()
					v, err := cache.Get(context.Background(), k)
					assert.NoError(t, err)
					assert.Equal(t, "result-"+k, v)
					// assert t=2000ms
					assert.InDelta(t, 2000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
				}()
			}
			wg.Wait()
			// assert replaceFn was triggered exactly 11 times
			assert.EqualValues(t, 11, cnt)
			// assert t=2000ms
			assert.InDelta(t, 2000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}
