# sc

[![GitHub release](https://img.shields.io/github/release/motoki317/sc.svg)](https://github.com/motoki317/sc/releases/)
![CI main](https://github.com/motoki317/sc/actions/workflows/main.yaml/badge.svg)
[![codecov](https://codecov.io/gh/motoki317/sc/branch/master/graph/badge.svg)](https://codecov.io/gh/motoki317/sc)
[![Go Reference](https://pkg.go.dev/badge/github.com/motoki317/sc.svg)](https://pkg.go.dev/github.com/motoki317/sc)

sc is a simple in-memory caching layer for golang.

[Introduction slide](https://speakerdeck.com/motoki317/effective-generic-cache-in-golang) (Japanese)

## Usage

Wrap your function with `sc`. It will automatically cache the returned values for a specified amount of time, with minimal overhead.

```go
import (
	"context"
	"fmt"
	"time"

	"github.com/motoki317/sc"
)

// HeavyData represents some data that is expensive to compute or fetch.
type HeavyData struct {
	Data string
	// ... and potentially many other fields
}

// retrieveHeavyData is the function that we want to cache.
// It simulates fetching data from a slow data source (e.g., a database or external API).
// The first argument must be context.Context.
// The second argument is the cache key (string in this example), which is generic.
// The return type is a pointer to HeavyData (which is also generic) and an error.
func retrieveHeavyData(_ context.Context, name string) (*HeavyData, error) {
	fmt.Printf("retrieveHeavyData called for key: %s\n", name) // To demonstrate when it's called
	// Simulate a slow operation
	time.Sleep(100 * time.Millisecond)
	return &HeavyData{
		Data: "my-data-for-" + name,
	}, nil
}

func main() {
	// Create a new cache instance:
	// - string: The type of the cache key.
	// - *HeavyData: The type of the value to be cached.
	// - retrieveHeavyData: The function to call when a cache miss occurs.
	// - 1*time.Minute: freshFor - How long the item is considered fresh.
	//                    During this period, Get() returns the cached value directly.
	// - 2*time.Minute: ttl - Time To Live. Overall duration an item remains in the cache.
	//                    If freshFor < ttl, after freshFor has passed (but before ttl expires),
	//                    Get() will return the stale data and trigger a background refresh.
	// - sc.WithLRUBackend(500): Optional. Specifies the cache backend.
	//                             Here, an LRU cache with a capacity of 500 items is used.
	//                             The default is an unbounded map-based cache.
	cache, err := sc.New[string, *HeavyData](
		retrieveHeavyData,
		1*time.Minute, // freshFor
		2*time.Minute, // ttl
		sc.WithLRUBackend(500), // Use LRU cache with capacity 500
	)
	if err != nil {
		panic(err)
	}

	// --- First call to Get ---
	// The cache is empty for key "foo", so retrieveHeavyData will be called.
	fmt.Println("Requesting 'foo' for the first time...")
	foo, err := cache.Get(context.Background(), "foo")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got foo: %+v\n", foo)

	// --- Second call to Get ---
	// "foo" is now in the cache and is fresh, so retrieveHeavyData will NOT be called.
	fmt.Println("\nRequesting 'foo' again (should be cached)...")
	foo, err = cache.Get(context.Background(), "foo")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got foo again: %+v\n", foo)

	// --- Example for a different key ---
	fmt.Println("\nRequesting 'bar' for the first time...")
	bar, err := cache.Get(context.Background(), "bar")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got bar: %+v\n", bar)

	// Wait for freshFor (1 min) + a bit, but less than ttl (2 min).
	// This timing helps demonstrate behavior around freshFor/ttl boundaries.
	// For this specific example, it mostly shows that after 1 min, the item is still cached.
	fmt.Println("\nWaiting for 1 minute and 5 seconds...")
	time.Sleep(1*time.Minute + 5*time.Second)

	// "foo" is now stale (past freshFor), but still within ttl.
	// If freshFor were shorter than ttl (as it is here: 1 min < 2 min), Get() on a stale item
	// returns the stale data and triggers a background refresh.
	// With freshFor (1 min) < ttl (2 min), graceful replacement is active.
	// The exact timing of the sleep might not always coincide with observing the
	// "retrieveHeavyData called..." print from a background refresh in this demo,
	// but the mechanism is in place. The key point is that data remains available.
	fmt.Println("\nRequesting 'foo' after 1 min 5 sec (graceful refresh might occur if not already updated)...")
	foo, err = cache.Get(context.Background(), "foo")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got foo after wait: %+v\n", foo)
	// If retrieveHeavyData was called again for "foo" above, it means a background refresh happened.
}

```

For a more detailed guide, including other backend options and advanced configurations, see the [Go Reference](https://pkg.go.dev/github.com/motoki317/sc).

## Notable Features

sc offers a range of features designed for simplicity, robustness, and performance:

**Ease of Use & Idiomatic Design:**
- **Simple API:** Wrap your function with `New()` and retrieve values with `Get()`.
- **No `Set()` Method:** `Get()` automatically handles value retrieval, promoting an idiomatic design that prevents [cache stampede](https://en.wikipedia.org/wiki/Cache_stampede) by design (see "Why no Set() method?" section for details).

**Robustness & Modern Go Features:**
- **Generics Support:** Leverages Go 1.18 generics for type safety for both keys and values, avoiding `interface{}` or `any` in internal implementations beyond type parameters.
- **Concurrency Safety:** All methods are safe for concurrent use from multiple goroutines.

**Performance & Concurrency Control:**
- **Single Flight Execution:** Ensures only one goroutine is launched per key to fetch the value, preventing redundant work.
- **Graceful Cache Replacement:** Allows serving stale data while a single background goroutine re-fetches a fresh value (when `freshFor` < `ttl`). This minimizes latency spikes.
- **Strict Request Coalescing:** Offers an option (`EnableStrictCoalescing()`) to ensure all callers receive fresh data, suitable for specific use-cases.

## Supported cache backends (cache replacement policy)

- Built-in map (default)
  - Note: This backend cannot have max number of items configured. It holds all values in memory until expiration. For more, see the [documentation](https://pkg.go.dev/github.com/motoki317/sc#WithMapBackend).
- LRU (Least Recently Used)
- 2Q (Two Queue Cache)

## The design

### Why no `Set()` method? The "Cache Layer" Philosophy

**The Core Idea:** `sc` is intentionally designed as a **"cache layer"** that sits seamlessly between your application and data source, rather than a general-purpose **"cache library"** that requires manual management. This distinction is key to its simplicity and robustness.

**`sc` as a Cache Layer:**
You provide `sc` with a function that knows how to fetch your data. From then on, you simply call `cache.Get()`. `sc` takes care of:
- Calling your function to get the data if it's not cached or is stale.
- Storing the data.
- Returning the cached data on subsequent calls.
- Automatically preventing issues like cache stampede (multiple, simultaneous fetches for the same data).

**The Problem with a Manual `Set()` Method:**
Traditional cache libraries often provide `Get()` and `Set()` methods. A typical workflow might look like this:
1. Try to `Get()` data from the cache.
2. If not found (cache miss), fetch data from the source.
3. `Set()` the fetched data into the cache.

While this offers flexibility, it also introduces potential pitfalls, especially in concurrent applications:
- **Cache Stampede:** Without careful locking, multiple requests experiencing a cache miss might all try to fetch and set the data simultaneously, overwhelming the data source.
- **Key Mismatches:** Developers might accidentally use different keys for `Get()` and `Set()`, leading to inconsistent caching.
- **Inconsistent Data Loading:** Logic for fetching data might be scattered or duplicated if not centralized.

**`sc`'s Solution: No `Set()` by Design**
By omitting a `Set()` method and requiring the data-fetching logic upfront (during cache instance creation), `sc` inherently avoids these problems:
- **Built-in Cache Stampede Prevention:** `sc` manages data retrieval, ensuring only one fetch operation occurs per key at any given time.
- **Guaranteed Key Consistency:** The same key used for `Get()` is used for the internal data retrieval function.
- **Centralized Data Fetching Logic:** Your data retrieval logic is defined once, making it easier to manage and reason about.

This design makes `sc` a "foolproof" cache layer: it handles the complexities of caching for you, reducing the likelihood of common caching-related bugs.

### What if I need to update or invalidate cached data?

`sc` operates as a "no-write-allocate" cache. This means your application should:
1. Update the original data in your primary data store (e.g., database).
2. Tell `sc` to remove the old data from the cache by calling `cache.Forget(key)`.

The next time `cache.Get(key)` is called for that item, `sc` will automatically fetch the updated data from your data source using the function you provided at setup.

This approach keeps data consistency clear: your data store is the source of truth, and `sc` is a performance layer that reflects it. Attempting to `Set()` data directly into the cache that differs from the data source could lead to inconsistencies. `sc`'s design prioritizes simplicity and predictability.
## Acknowledgements

I would like to thank the following libraries for giving me ideas:

- [go-chi/stampede: Function and HTTP request coalescer](https://github.com/go-chi/stampede)
  - For "request coalescing" and "cache layer" idea
- [singleflight package - golang.org/x/sync/singleflight - pkg.go.dev](https://pkg.go.dev/golang.org/x/sync/singleflight)
  - For internal implementation
- [Songmu/smartcache](https://github.com/Songmu/smartcache)
  - For "graceful replacement" idea
    - The term "graceful" comes from the [varnish](https://varnish-cache.org/) configuration.
- [methane/zerotimecache](https://github.com/methane/zerotimecache)
  - For "zero-time cache" idea
