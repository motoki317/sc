package sc

import (
	"context"
	"errors"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/motoki317/sc/lru"
	"github.com/motoki317/sc/tq"
)

func TestNewMust(t *testing.T) {
	t.Parallel()

	fn := func(ctx context.Context, s string) (string, error) { return "", nil }

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		_ = NewMust(fn, 0, 0)
	})
	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		func() {
			defer func() {
				err := recover()
				assert.NotNil(t, err)
			}()

			_ = NewMust(fn, -1, -1)
		}()
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	fn := func(ctx context.Context, s string) (string, error) { return "", nil }

	t.Run("defaults to map", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0)
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, mapBackend[string, value[string]]{}, c.values)
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

		c, err := New[string, string](fn, 0, 0, WithMapBackend(0))
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, mapBackend[string, value[string]]{}, c.values)
		assert.False(t, c.strictCoalescing)
	})

	t.Run("map cache with invalid capacity", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 0, 0, WithMapBackend(-1))
		assert.Error(t, err)
	})

	t.Run("map cache with capacity", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, WithMapBackend(10))
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, mapBackend[string, value[string]]{}, c.values)
		assert.False(t, c.strictCoalescing)
	})

	t.Run("strict map cache", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, WithMapBackend(0), EnableStrictCoalescing())
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, mapBackend[string, value[string]]{}, c.values)
		assert.True(t, c.strictCoalescing)
	})

	t.Run("strict map cache with capacity", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, WithMapBackend(10), EnableStrictCoalescing())
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, mapBackend[string, value[string]]{}, c.values)
		assert.True(t, c.strictCoalescing)
	})

	t.Run("LRU needs capacity set", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 0, 0, WithLRUBackend(0))
		assert.Error(t, err)
	})

	t.Run("LRU cache with invalid capacity", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 0, 0, WithLRUBackend(-1))
		assert.Error(t, err)
	})

	t.Run("struct LRU needs capacity set", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 0, 0, WithLRUBackend(-1), EnableStrictCoalescing())
		assert.Error(t, err)
	})

	t.Run("LRU cache", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, WithLRUBackend(10))
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, &lru.Cache[string, value[string]]{}, c.values)
		assert.False(t, c.strictCoalescing)
	})

	t.Run("strict LRU cache", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, WithLRUBackend(10), EnableStrictCoalescing())
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, &lru.Cache[string, value[string]]{}, c.values)
		assert.True(t, c.strictCoalescing)
	})

	t.Run("2Q needs capacity set", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 0, 0, With2QBackend(0))
		assert.Error(t, err)
	})

	t.Run("2Q cache with invalid capacity", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 0, 0, With2QBackend(-1))
		assert.Error(t, err)
	})

	t.Run("struct 2Q needs capacity set", func(t *testing.T) {
		t.Parallel()

		_, err := New[string, string](fn, 0, 0, With2QBackend(0), EnableStrictCoalescing())
		assert.Error(t, err)
	})

	t.Run("2Q cache", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, With2QBackend(10))
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, &tq.Cache[string, value[string]]{}, c.values)
		assert.False(t, c.strictCoalescing)
	})

	t.Run("strict 2Q cache", func(t *testing.T) {
		t.Parallel()

		c, err := New[string, string](fn, 0, 0, With2QBackend(10), EnableStrictCoalescing())
		assert.NoError(t, err)
		assert.IsType(t, &Cache[string, string]{}, c)
		assert.IsType(t, &tq.Cache[string, value[string]]{}, c.values)
		assert.True(t, c.strictCoalescing)
	})
}

// TestCache_Get calls Cache.Get multiple times and ensures a value is reused.
func TestCache_Get(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
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

// TestCache_Get_Async ensures that Cache.Get will trigger background fetch if a stale value is found.
func TestCache_Get_Async(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
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
					val, err := cache.Get(context.Background(), "k1")
					assert.NoError(t, err)
					assert.Equal(t, "result1", val)
				}()
			}
			wg.Wait()
			assert.EqualValues(t, 1, atomic.LoadInt64(&cnt))
			// assert t=500ms
			assert.InDelta(t, 500*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))

			// t=500ms, 2nd call group -> returns stale values, one goroutine is launched in the background to trigger replaceFn
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					val, err := cache.Get(context.Background(), "k1")
					assert.NoError(t, err)
					assert.Equal(t, "result1", val)
				}()
			}
			wg.Wait()
			// assert t=500ms
			assert.InDelta(t, 500*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			// Sleep for some time to make sure the background goroutine triggers replaceFn
			time.Sleep(250 * time.Millisecond)
			assert.EqualValues(t, 2, atomic.LoadInt64(&cnt))
		})
	}
}

// TestCache_Get_Error ensures Cache.Get returns an error if replaceFn returns an error.
func TestCache_Get_Error(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
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

// TestCache_Forget_Interrupt ensures that calling Cache.Forget will make later get calls to trigger replaceFn.
func TestCache_Forget_Interrupt(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
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

// TestCache_Forget_NoInterrupt is similar to TestCache_Forget_Interrupt, but there are no ongoing calls of replaceFn.
func TestCache_Forget_NoInterrupt(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
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

// TestCache_Purge_Interrupt ensures that calling Cache.Purge will make all later get calls to trigger replaceFn.
func TestCache_Purge_Interrupt(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var cnt int64
			replaceFn := func(ctx context.Context, key string) (string, error) {
				atomic.AddInt64(&cnt, 1)
				time.Sleep(750 * time.Millisecond)
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, c.cacheOpts...)
			assert.NoError(t, err)

			t0 := time.Now()
			var wg sync.WaitGroup
			// t=0ms, 1st call
			wg.Add(2)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "result-k1", v)
			}()
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k2")
				assert.NoError(t, err)
				assert.Equal(t, "result-k2", v)
			}()
			time.Sleep(500 * time.Millisecond)
			// t=500ms, Purge, then 2nd call
			cache.Purge()
			wg.Add(2)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "result-k1", v)
			}()
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k2")
				assert.NoError(t, err)
				assert.Equal(t, "result-k2", v)
			}()
			wg.Wait()
			// t=1250ms, assert replaceFn was triggered 4 times
			assert.EqualValues(t, 4, cnt)
			assert.InDelta(t, 1250*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}

// TestCache_Purge_NoInterrupt is similar to TestCache_Purge_Interrupt, but there are no ongoing calls of replaceFn.
func TestCache_Purge_NoInterrupt(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var cnt int64
			replaceFn := func(ctx context.Context, key string) (string, error) {
				atomic.AddInt64(&cnt, 1)
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, c.cacheOpts...)
			assert.NoError(t, err)

			// 1st call group
			v, err := cache.Get(context.Background(), "k1")
			assert.NoError(t, err)
			assert.Equal(t, "result-k1", v)
			assert.EqualValues(t, cnt, 1)
			v, err = cache.Get(context.Background(), "k2")
			assert.NoError(t, err)
			assert.Equal(t, "result-k2", v)
			assert.EqualValues(t, cnt, 2)

			// 2nd call group - values are reused
			v, err = cache.Get(context.Background(), "k1")
			assert.NoError(t, err)
			assert.Equal(t, "result-k1", v)
			assert.EqualValues(t, cnt, 2)
			v, err = cache.Get(context.Background(), "k2")
			assert.NoError(t, err)
			assert.Equal(t, "result-k2", v)
			assert.EqualValues(t, cnt, 2)

			cache.Purge()

			// 3rd call group - all values are forgotten
			v, err = cache.Get(context.Background(), "k1")
			assert.NoError(t, err)
			assert.Equal(t, "result-k1", v)
			assert.EqualValues(t, cnt, 3)
			v, err = cache.Get(context.Background(), "k2")
			assert.NoError(t, err)
			assert.Equal(t, "result-k2", v)
			assert.EqualValues(t, cnt, 4)
		})
	}
}

// TestCache_ParallelReplacement ensures parallel call to replaceFn per key, not per cache instance.
func TestCache_ParallelReplacement(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
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
			wg.Add(2)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "result-k1", v)
			}()
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k2")
				assert.NoError(t, err)
				assert.Equal(t, "result-k2", v)
			}()
			wg.Wait()
			// t=500ms, assert replaceFn was triggered twice
			assert.EqualValues(t, 2, cnt)
			// assert t=500ms
			assert.InDelta(t, 500*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}

// TestCache_MultipleValues calls Cache.Get with some different keys, and ensures correct values are returned.
func TestCache_MultipleValues(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
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

// TestCache_NoStrictCoalescing tests "no strict coalescing" behavior, which is similar to singleflight.
// "No strict coalescing" cache may return expired values.
func TestCache_NoStrictCoalescing(t *testing.T) {
	t.Parallel()

	for _, c := range nonStrictCaches(10) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var cnt int64
			replaceFn := func(ctx context.Context, key string) (string, error) {
				atomic.AddInt64(&cnt, 1)
				assert.Equal(t, "k1", key)
				time.Sleep(1 * time.Second)
				return "value1", nil
			}
			cache, err := New[string, string](replaceFn, 500*time.Millisecond, 500*time.Millisecond, c.cacheOpts...)
			assert.NoError(t, err)

			t0 := time.Now()
			var wg sync.WaitGroup
			// t=0ms, 1st call -> triggers replaceFn
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "value1", v)
				t.Log("1st call return")
				// assert t=1000ms
				assert.InDelta(t, 1000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}()
			time.Sleep(250 * time.Millisecond)
			// t=250ms, assert replaceFn was called only once
			assert.EqualValues(t, 1, atomic.LoadInt64(&cnt))
			// t=250ms, 2nd call -> should not trigger replaceFn, to be coalesced with the 1st call
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "value1", v)
				t.Log("2nd call return")
				// assert t=250ms
				assert.InDelta(t, 1000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}()
			time.Sleep(500 * time.Millisecond)
			// t=750ms, assert replaceFn was called only once
			assert.EqualValues(t, 1, atomic.LoadInt64(&cnt))
			// t=750ms, 3rd call -> returns stale value, to be coalesced with the 1st and 2nd call
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "value1", v)
				t.Log("3rd call return")
				// assert t=1000ms
				assert.InDelta(t, 1000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}()
			time.Sleep(500 * time.Millisecond)
			wg.Wait()
			// assert t=1250ms
			assert.InDelta(t, 1250*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			// t=1250ms, assert replaceFn was called only once
			assert.EqualValues(t, 1, atomic.LoadInt64(&cnt))
			// t=1250ms, 4th call -> should trigger replaceFn
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "value1", v)
				t.Log("4th call return")
				// assert t=2250ms
				assert.InDelta(t, 2250*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}()
			time.Sleep(1250 * time.Millisecond)
			wg.Wait()
			// t=2500ms, all calls should have finished
			assert.EqualValues(t, 2, atomic.LoadInt64(&cnt))
			// assert t=2500ms
			assert.InDelta(t, 2500*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}

// TestCache_StrictCoalescing ensures "strict coalescing" cache will never return expired items.
func TestCache_StrictCoalescing(t *testing.T) {
	t.Parallel()

	for _, c := range strictCaches(10) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var cnt int64
			replaceFn := func(ctx context.Context, key string) (string, error) {
				atomic.AddInt64(&cnt, 1)
				assert.Equal(t, "k1", key)
				time.Sleep(1 * time.Second)
				return "value1", nil
			}
			cache, err := New[string, string](replaceFn, 500*time.Millisecond, 500*time.Millisecond, c.cacheOpts...)
			assert.NoError(t, err)

			t0 := time.Now()
			var wg sync.WaitGroup
			// t=0ms, 1st call -> triggers replaceFn
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "value1", v)
				t.Log("1st call return")
				// assert t=1000ms
				assert.InDelta(t, 1000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}()
			time.Sleep(250 * time.Millisecond)
			// t=250ms, assert replaceFn was called only once
			assert.EqualValues(t, 1, atomic.LoadInt64(&cnt))
			// t=250ms, 2nd call -> should not trigger replaceFn, to be coalesced with the 1st call
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "value1", v)
				t.Log("2nd call return")
				// assert t=1000ms
				assert.InDelta(t, 1000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}()
			time.Sleep(500 * time.Millisecond)
			// t=750ms, assert replaceFn was called only once
			assert.EqualValues(t, 1, atomic.LoadInt64(&cnt))
			// t=750ms, 3rd call -> should trigger replaceFn after the first call returns
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "value1", v)
				t.Log("3rd call return")
				// assert t=2000ms
				assert.InDelta(t, 2000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}()
			time.Sleep(500 * time.Millisecond)
			// t=1250ms, assert replaceFn was called twice
			assert.EqualValues(t, 2, atomic.LoadInt64(&cnt))
			// t=1250ms, 4th call -> should be coalesced with the 3rd call
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "value1", v)
				t.Log("4th call return")
				// assert t=2000ms
				assert.InDelta(t, 2000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}()
			time.Sleep(1 * time.Second)
			wg.Wait()
			// t=2250ms, all calls should have finished
			assert.EqualValues(t, 2, atomic.LoadInt64(&cnt))
			// assert t=2250ms
			assert.InDelta(t, 2250*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}

// TestCache_ZeroTimeCache ensures "strict coalescing" cache will never return expired items, even with zero freshFor/ttl values.
func TestCache_ZeroTimeCache(t *testing.T) {
	t.Parallel()

	for _, c := range strictCaches(10) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var cnt int64
			replaceFn := func(ctx context.Context, key string) (string, error) {
				atomic.AddInt64(&cnt, 1)
				assert.Equal(t, "k1", key)
				time.Sleep(1 * time.Second)
				return "value1", nil
			}
			cache, err := New[string, string](replaceFn, 0, 0, c.cacheOpts...)
			assert.NoError(t, err)

			t0 := time.Now()
			var wg sync.WaitGroup
			// t=0ms, 1st call -> triggers replaceFn
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "value1", v)
				t.Log("1st call return")
				// assert t=1000ms
				assert.InDelta(t, 1000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}()
			time.Sleep(250 * time.Millisecond)
			// t=250ms, assert replaceFn was called only once
			assert.EqualValues(t, 1, atomic.LoadInt64(&cnt))
			// t=250ms, 2nd call -> should NOT be coalesced with the 1st call
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "value1", v)
				t.Log("2nd call return")
				// assert t=2000ms
				assert.InDelta(t, 2000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}()
			time.Sleep(500 * time.Millisecond)
			// t=750ms, assert replaceFn was called only once
			assert.EqualValues(t, 1, atomic.LoadInt64(&cnt))
			// t=750ms, 3rd call -> should be coalesced with the 2nd call
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "value1", v)
				t.Log("3rd call return")
				// assert t=2000ms
				assert.InDelta(t, 2000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}()
			time.Sleep(500 * time.Millisecond)
			// t=1250ms, assert replaceFn was called twice
			assert.EqualValues(t, 2, atomic.LoadInt64(&cnt))
			// t=1250ms, 4th call -> should NOT be coalesced with the 3rd call
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := cache.Get(context.Background(), "k1")
				assert.NoError(t, err)
				assert.Equal(t, "value1", v)
				t.Log("4th call return")
				// assert t=3000ms
				assert.InDelta(t, 3000*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
			}()
			time.Sleep(2 * time.Second)
			wg.Wait()
			// t=3250ms, all calls should have finished
			assert.EqualValues(t, 3, atomic.LoadInt64(&cnt))
			// assert t=3250ms
			assert.InDelta(t, 3250*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}

// TestCleaningCache tests caches with cleaner option, which will clean up expired items on a regular interval.
func TestCleaningCache(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var cnt int64
			replaceFn := func(ctx context.Context, key string) (string, error) {
				atomic.AddInt64(&cnt, 1)
				return "value-" + key, nil
			}
			cache, err := New(replaceFn, 700*time.Millisecond, 1000*time.Millisecond, append(c.cacheOpts, WithCleanupInterval(300*time.Millisecond))...)
			assert.NoError(t, err)

			// t=0ms, cache the value
			v, err := cache.Get(context.Background(), "k1")
			assert.NoError(t, err)
			assert.Equal(t, "value-k1", v)
			assert.EqualValues(t, 1, atomic.LoadInt64(&cnt))

			time.Sleep(400 * time.Millisecond)
			// t=400ms, value is still cached and fresh
			v, err = cache.Get(context.Background(), "k1")
			assert.NoError(t, err)
			assert.Equal(t, "value-k1", v)
			assert.EqualValues(t, 1, atomic.LoadInt64(&cnt))

			time.Sleep(1 * time.Second)
			// t=1400ms, expired value is automatically removed from the cache, freeing memory
			// although, this has no effect if viewed from the public interface of Cache
			v, err = cache.Get(context.Background(), "k1")
			assert.NoError(t, err)
			assert.Equal(t, "value-k1", v)
			assert.EqualValues(t, 2, atomic.LoadInt64(&cnt))
		})
	}
}

// TestCleaningCacheFinalizer tests that cache finalizers to stop cleaner is working.
// Since there's not really a good way of ensuring call to the finalizer, this just increases the test coverage.
func TestCleaningCacheFinalizer(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			replaceFn := func(_ context.Context, _ struct{}) (string, error) { return "", nil }
			c, err := New(replaceFn, time.Hour, time.Hour, append(c.cacheOpts, WithCleanupInterval(time.Second))...)
			assert.NoError(t, err)

			_, _ = c.Get(context.Background(), struct{}{})
			runtime.GC() // finalizer is called and cleaner is stopped
		})
	}
}
