# sc

[![GitHub release](https://img.shields.io/github/release/motoki317/sc.svg)](https://github.com/motoki317/sc/releases/)
![CI main](https://github.com/motoki317/sc/actions/workflows/main.yaml/badge.svg)
[![codecov](https://codecov.io/gh/motoki317/sc/branch/master/graph/badge.svg)](https://codecov.io/gh/motoki317/sc)
[![Go Reference](https://pkg.go.dev/badge/github.com/motoki317/sc.svg)](https://pkg.go.dev/github.com/motoki317/sc)

sc is a simple in-memory caching layer for golang.

## Usage

Wrap your function with sc - it will automatically cache the values for specified amount of time, with minimal overhead.

```go
type HeavyData struct {
	Data string
	// and all the gazillion fields you may have in your data
}

func retrieveHeavyData(_ context.Context, name string) (*HeavyData, error) {
	// Query to database or something...
	return &HeavyData{
		Data: "my-data-" + name,
	}, nil
}

func main() {
	// Wrap your data retrieval function.
	cache, _ := sc.New[string, *HeavyData](retrieveHeavyData, 1*time.Minute, 2*time.Minute, sc.WithLRUBackend(500))
	// It will automatically call the given function if value is missing.
	foo, _ := cache.Get(context.Background(), "foo")
}
```

For a more detailed guide, see [reference](https://pkg.go.dev/github.com/motoki317/sc).

## Notable Features

- Simple to use: wrap your function with `New()` and just call `Get()`.
    - There is no `Set()` method. Calling `Get()` will automatically retrieve the value for you.
    - This prevents [cache stampede](https://en.wikipedia.org/wiki/Cache_stampede) problem idiomatically (see below).
- Supports 1.18 generics - both key and value are generic.
    - No `interface{}` or `any` used other than in type parameters, even in internal implementations.
- All methods are safe to be called from multiple goroutines.
- Allows 'graceful cache replacement' (if `freshFor` < `ttl`) - a single goroutine is launched in the background to
  re-fetch a fresh value while serving stale value to readers.
- Allows strict request coalescing (`EnableStrictCoalescing()` option) - ensures that all returned values are fresh (a
  niche use-case).

## Supported cache backends (cache replacement policy)

The default backend is the built-in map.
This is ultra-lightweight, but does **not** evict items.
You should only use the built-in map backend if your key's cardinality is finite,
and you are comfortable holding **all** values in-memory.

Otherwise, you should use LRU or 2Q backend which automatically evicts overflown items.

- Built-in map (default)
- LRU (Least Recently Used)
- 2Q (Two Queue Cache)

## The design

### Why no Set() method? / Why cannot I dynamically provide load function to Get() method?

Short answer: sc is designed as a foolproof 'cache layer', not an overly complicated 'cache library'.

Long answer:

sc is designed as a simple, foolproof 'cache layer'.
Users of sc simply wrap data-retrieving functions and retrieve values via the cache.
By doing so, sc automatically reuses retrieved values and minimizes load on your data-store.

Now, let's imagine how users would use a more standard cache library with `Set()` method.
One could use `Get()` and `Set()` method to build the following logic:

1. `Get()` from the cache.
2. If the value is not in the cache, retrieve it from the source.
3. `Set()` the value.

This is probably the most common use-case, and it is fine for most applications.
But if you do not write it properly, the following problems may occur:

- If data flow is large, cache stampede might occur.
- Accidentally using different keys for `Get()` and `Set()`.
- Over-caching or under-caching by using inappropriate keys.

sc solves the problems mentioned above by acting as a 'cache layer'.

- sc will manage the requests for you - no risk of accidentally writing a bad caching logic and overloading your data-store with cache stampede.
- No manual `Set()` needed - no risk of accidentally using different keys.
- Only the cache key is passed to the pre-provided replacement function - no risk of over-caching or under-caching.

This is why sc does not have a `Set()` method, and forces you to provide replacement function on setup.
In this way, there is no risk of cache stampede and possible bugs described above -
sc will handle it for you.

### But I still want to manually `Set()` value on update!

By the nature of the design, sc is a no-write-allocate type cache.
You update the value on the data-store, and then call `Forget()` to clear the value on the cache.
sc will automatically load the value next time `Get()` is called.

One could design another cache layer library with `Set()` method which automatically calls the pre-provided
update function which updates the data-store, then updates the value on the cache.
But that would add whole another level of complexity - sc aims to be a simple cache layer.

## Inspirations from

- [go-chi/stampede: Function and HTTP request coalescer](https://github.com/go-chi/stampede)
- [singleflight package - golang.org/x/sync/singleflight - pkg.go.dev](https://pkg.go.dev/golang.org/x/sync/singleflight)
- [methane/zerotimecache](https://github.com/methane/zerotimecache)
