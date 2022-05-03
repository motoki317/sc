package sc

import (
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// Filter iterates over elements of collection, returning an array of all elements predicate returns truthy for.
// Copied from github.com/samber/lo.
func Filter[V any](collection []V, predicate func(V, int) bool) []V {
	var result []V

	for i, item := range collection {
		if predicate(item, i) {
			result = append(result, item)
		}
	}

	return result
}

// Map manipulates a slice and transforms it to a slice of another type.
// Copied from github.com/samber/lo.
func Map[T any, R any](collection []T, iteratee func(T, int) R) []R {
	result := make([]R, len(collection))

	for i, item := range collection {
		result[i] = iteratee(item, i)
	}

	return result
}

type testCase struct {
	name      string
	cacheOpts []CacheOption
}

func nonStrictCaches(cap int) []testCase {
	return []testCase{
		{name: "map cache", cacheOpts: []CacheOption{WithMapBackend(cap)}},
		{name: "LRU cache", cacheOpts: []CacheOption{WithLRUBackend(cap)}},
		{name: "2Q cache", cacheOpts: []CacheOption{With2QBackend(cap)}},
	}
}

func strictCaches(cap int) []testCase {
	return Map(nonStrictCaches(cap), func(t testCase, _ int) testCase {
		return testCase{
			name:      "strict " + t.name,
			cacheOpts: append(t.cacheOpts, EnableStrictCoalescing()),
		}
	})
}

func allCaches(cap int) []testCase {
	return append(nonStrictCaches(cap), strictCaches(cap)...)
}

func evictingCaches(cap int) []testCase {
	return Filter(allCaches(cap), func(c testCase, _ int) bool { return !strings.HasSuffix(c.name, "map cache") })
}

func newZipfian(s, v float64, size uint64) func() string {
	zipf := rand.NewZipf(rand.New(rand.NewSource(time.Now().UnixNano())), s, v, size)
	return func() string {
		return strconv.Itoa(int(zipf.Uint64()))
	}
}

func newKeys(next func() string, size int) []string {
	keys := make([]string, size)
	for i := 0; i < size; i++ {
		keys[i] = next()
	}
	return keys
}
