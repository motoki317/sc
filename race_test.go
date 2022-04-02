//go:build !race

package sc

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache_BackGroundFetch(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		cacheOpts []CacheOption
	}{
		{name: "map cache", cacheOpts: []CacheOption{WithMapBackend()}},
		{name: "strict map cache", cacheOpts: []CacheOption{WithMapBackend(), EnableStrictCoalescing()}},
		{name: "LRU cache", cacheOpts: []CacheOption{WithLRUBackend(10)}},
		{name: "strict LRU cache", cacheOpts: []CacheOption{WithLRUBackend(10), EnableStrictCoalescing()}},
	}

	for _, c := range cases {
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
			assert.EqualValues(t, 1, cnt)
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
			assert.EqualValues(t, 2, cnt) // NOTE: causes race condition on cnt
		})
	}
}

func TestCache_NoStrictCoalescing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		cacheOpts []CacheOption
	}{
		{name: "map cache", cacheOpts: []CacheOption{WithMapBackend()}},
		{name: "LRU cache", cacheOpts: []CacheOption{WithLRUBackend(10)}},
	}

	for _, c := range cases {
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
			assert.EqualValues(t, 1, cnt) // NOTE: causes race condition on cnt
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
			assert.EqualValues(t, 1, cnt)
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
			assert.EqualValues(t, 1, cnt)
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
			assert.EqualValues(t, 2, cnt)
			// assert t=2500ms
			assert.InDelta(t, 2500*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}

func TestCache_StrictCoalescing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		cacheOpts []CacheOption
	}{
		{name: "strict map cache", cacheOpts: []CacheOption{WithMapBackend(), EnableStrictCoalescing()}},
		{name: "strict LRU cache", cacheOpts: []CacheOption{WithLRUBackend(10), EnableStrictCoalescing()}},
	}

	for _, c := range cases {
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
			assert.EqualValues(t, 1, cnt) // NOTE: causes race condition on cnt
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
			assert.EqualValues(t, 1, cnt)
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
			assert.EqualValues(t, 2, cnt)
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
			assert.EqualValues(t, 2, cnt)
			// assert t=2250ms
			assert.InDelta(t, 2250*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}

func TestCache_ZeroTimeCache(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		cacheOpts []CacheOption
	}{
		{name: "strict map cache", cacheOpts: []CacheOption{WithMapBackend(), EnableStrictCoalescing()}},
		{name: "strict LRU cache", cacheOpts: []CacheOption{WithLRUBackend(10), EnableStrictCoalescing()}},
	}

	for _, c := range cases {
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
			assert.EqualValues(t, 1, cnt) // NOTE: causes race condition on cnt
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
			assert.EqualValues(t, 1, cnt)
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
			assert.EqualValues(t, 2, cnt)
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
			assert.EqualValues(t, 3, cnt)
			// assert t=3250ms
			assert.InDelta(t, 3250*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}
