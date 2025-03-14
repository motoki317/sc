package sc

import (
	"runtime"
	"time"
	"weak"
)

// cleaner is launched as a single goroutine to regularly clean up expired items from the cache.
// cleaner holds reference to cache, not Cache - this allows finalizers to be run on Cache.
//
// See https://github.com/patrickmn/go-cache/blob/46f407853014144407b6c2ec7ccc76bf67958d93/cache.go#L1115 for more on this design.
type cleaner[K comparable, V any] struct {
	closer chan struct{}
	// We use weak pointer here in order to deal with an extremely-unlikely case where cached data itself
	// somehow has a reference to *Cache itself, forming a reference cycle.
	// If above is the case, and we're using strong reference here, the cleaner goroutine keeps a reference to this
	// reference cycle, therefore *Cache and *cache never being evicted, leading to memory leaks.
	//
	// This case can be tested in TestCleaningCacheFinalizer, where testers can manually check that stopCleaner function
	// is called, for example by adding `fmt.Println("cleanup called")` there.
	c weak.Pointer[cache[K, V]]
}

func startCleaner[K comparable, V any](c *Cache[K, V], interval time.Duration) {
	closer := make(chan struct{})
	cl := &cleaner[K, V]{
		closer: closer,
		c:      weak.Make(c.cache),
	}
	go cl.run(interval)
	runtime.AddCleanup(c, stopCleaner, closer)
}

func (cl *cleaner[K, V]) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c := cl.c.Value()
			if c == nil {
				return
			}
			c.cleanup()
		case <-cl.closer:
			return
		}
	}
}

func stopCleaner(closer chan<- struct{}) {
	close(closer)
}
