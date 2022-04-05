# sc

[![GitHub release](https://img.shields.io/github/release/motoki317/sc.svg)](https://github.com/motoki317/sc/releases/)
![CI main](https://github.com/motoki317/sc/actions/workflows/main.yaml/badge.svg)
[![codecov](https://codecov.io/gh/motoki317/sc/branch/master/graph/badge.svg)](https://codecov.io/gh/motoki317/sc)
[![Go Reference](https://pkg.go.dev/badge/github.com/motoki317/sc.svg)](https://pkg.go.dev/github.com/motoki317/sc)

sc is a simple golang in-memory caching library, with easily configurable implementations.

## Notable Features

- Simple to use; the only methods are Get(), GetFresh(), and Forget().
  - There is no Set() method - this is an intentional design choice to make the use easier.
- Supports 1.18 generics - both key and value are generic.
  - No `interface{}` even in internal implementations.
- Supports multiple cache backends.
- Prevents [cache stampede](https://en.wikipedia.org/wiki/Cache_stampede) problem idiomatically.
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

## Usage

See [reference](https://pkg.go.dev/github.com/motoki317/sc).

## Borrowed Ideas

- [go-chi/stampede: Function and HTTP request coalescer](https://github.com/go-chi/stampede)
- [singleflight package - golang.org/x/sync/singleflight - pkg.go.dev](https://pkg.go.dev/golang.org/x/sync/singleflight)
