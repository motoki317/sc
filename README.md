# sc

[![GitHub release](https://img.shields.io/github/release/motoki317/sc.svg)](https://github.com/motoki317/sc/releases/)
![CI main](https://github.com/motoki317/sc/actions/workflows/main.yaml/badge.svg)
[![codecov](https://codecov.io/gh/motoki317/sc/branch/master/graph/badge.svg)](https://codecov.io/gh/motoki317/sc)
[![Go Reference](https://pkg.go.dev/badge/github.com/motoki317/sc.svg)](https://pkg.go.dev/github.com/motoki317/sc)

sc is a simple idiomatic in-memory caching library.

## Usage

See [reference](https://pkg.go.dev/github.com/motoki317/sc).

## Notable Features

- Simple to use: the only methods are `Get()`, `GetFresh()`, and `Forget()`.
  - There is no `Set()` method - this is an intentional design choice to make the use easier.
  - This prevents [cache stampede](https://en.wikipedia.org/wiki/Cache_stampede) problem idiomatically (see below).
- Supports 1.18 generics - both key and value are generic.
  - No `interface{}` or `any` used other than in the type parameter, even in internal implementations.
- All methods are safe to be called from multiple goroutines.
- Allows graceful cache replacement (if `freshFor` < `ttl`) - only one goroutine is launched in the background to re-fetch the value.
- Allows strict request coalescing (`EnableStrictCoalescing()` option) - ensures that all returned values are fresh (a niche use-case).

## Supported cache backends (cache replacement policy)

The default backend is the built-in map.
This is ultra-lightweight, but does **not** evict items.
You should only use the built-in map backend if your key's cardinality is finite,
and you are comfortable holding **all** values in-memory.

Otherwise, you should use LRU or ARC backend which automatically evicts overflown items.

- Built-in map (default)
- LRU (Least Recently Used)
- ARC (Adaptive Replacement Cache)

## Why no Set() method?

**tl;dr**: Most use-cases do not even need a manual `Set()`, so it is simpler this way.

While it would be easy to add Set() method to the `*Cache[K, V]` type, it is not there
for a reason.

sc prevents the 'cache stampede' problem without requiring users to write a complicated code -
if multiple goroutines call `Get()` method at the same time, sc automatically coalesces
requests and makes sure replaceFn is called by only one goroutine at any point for each
key value (unless `Forget()` is called on the key).
This is one of the core concepts of sc.

Now, let's imagine how users of a cache library would use `Set()` method.
One could use `Get()` and `Set()` method to build the following logic:

1. `Get()` from the cache.
2. If the value is not in the cache, retrieve it from the source.
3. `Set()` the value.

This is probably the most common use-case, and it is fine for most applications.
But if the data flow is large, this simple logic contains the risk of cache stampede
if you do not lock it properly.
What's more, writing such logic by yourself also contains risks like forgetting to call
`Set()`, accidentally using different keys for `Get()` and `Set()`, and so on.
Most users do not need to write cache logic by themselves.

This is why sc does not have a `Set()` method - users can just provide replaceFn on the setup,
then just use `Get()` to automatically access the source if necessary.
In this way, there is no risk of cache stampede and possible bugs described above -
sc will handle it for you.
If you still need to use `Set()` for some reason, then this library may not be for you.

Since there is no `Set()` method, users cannot set the value on write (no-write-allocate).
But for similar reasons described above, it is simpler this way and there is less risk of
probable bugs.

## Borrowed Ideas

- [go-chi/stampede: Function and HTTP request coalescer](https://github.com/go-chi/stampede)
- [singleflight package - golang.org/x/sync/singleflight - pkg.go.dev](https://pkg.go.dev/golang.org/x/sync/singleflight)
