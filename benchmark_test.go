package sc

import (
	"context"
	"testing"
	"time"
)

func BenchmarkCache_Map(b *testing.B) {
	replaceFn := func(ctx context.Context, key string) (string, error) {
		return "value", nil
	}
	cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, WithMapBackend())
	if err != nil {
		b.Error(err)
	}

	ctx := context.Background()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(ctx, "key")
	}
}

func BenchmarkCache_MapStrict(b *testing.B) {
	replaceFn := func(ctx context.Context, key string) (string, error) {
		return "value", nil
	}
	cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, WithMapBackend(), EnableStrictCoalescing())
	if err != nil {
		b.Error(err)
	}

	ctx := context.Background()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(ctx, "key")
	}
}

func BenchmarkCache_LRU(b *testing.B) {
	replaceFn := func(ctx context.Context, key string) (string, error) {
		return "value", nil
	}
	cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, WithLRUBackend(10))
	if err != nil {
		b.Error(err)
	}

	ctx := context.Background()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(ctx, "key")
	}
}

func BenchmarkCache_LRUStrict(b *testing.B) {
	replaceFn := func(ctx context.Context, key string) (string, error) {
		return "value", nil
	}
	cache, err := New[string, string](replaceFn, 1*time.Second, 1*time.Second, WithLRUBackend(10), EnableStrictCoalescing())
	if err != nil {
		b.Error(err)
	}

	ctx := context.Background()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(ctx, "key")
	}
}
