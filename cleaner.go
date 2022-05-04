package sc

import (
	"runtime"
	"time"
)

// cleaner is launched as a single goroutine to regularly clean up expired items from the cache.
// cleaner holds reference to cache, not Cache - this allows finalizers to be run on Cache.
//
// See https://github.com/patrickmn/go-cache/blob/46f407853014144407b6c2ec7ccc76bf67958d93/cache.go#L1115 for more on this design.
type cleaner[K comparable, V any] struct {
	closer chan struct{}
	c      *cache[K, V]
}

func startCleaner[K comparable, V any](c *Cache[K, V], interval time.Duration) {
	cl := &cleaner[K, V]{
		closer: make(chan struct{}),
		c:      c.cache,
	}
	go cl.run(interval)
	runtime.SetFinalizer(c, stopCleaner(cl))
}

func (cl *cleaner[K, V]) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cl.c.cleanup()
		case <-cl.closer:
			return
		}
	}
}

func (cl *cleaner[K, V]) stop() {
	cl.closer <- struct{}{}
}

func stopCleaner[K comparable, V any](cl *cleaner[K, V]) func(*Cache[K, V]) {
	return func(_ *Cache[K, V]) {
		cl.stop()
	}
}
