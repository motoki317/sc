package sc

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStats_String(t *testing.T) {
	tests := []struct {
		name  string
		stats Stats
		want  string
	}{
		{
			name: "simple",
			stats: Stats{
				HitStats{1, 2, 3, 4},
				SizeStats{5, 6},
			},
			want: "Hits: 1, GraceHits: 2, Misses: 3, Replacements: 4, Hit Ratio: 0.500000, Size: 5, Capacity: 6",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.stats.String(), "String()")
		})
	}
}

func TestStats_HitRatio(t *testing.T) {
	type fields struct {
		Hits         uint64
		GraceHits    uint64
		Misses       uint64
		Replacements uint64
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{
			name:   "simple",
			fields: fields{1, 2, 3, 4},
			want:   (1.0 + 2.0) / (1.0 + 2.0 + 3.0), // 0.5
		},
		{
			name:   "simple 2",
			fields: fields{123, 456, 789, 700},
			want:   (123.0 + 456.0) / (123.0 + 456.0 + 789.0), // 0.423245...
		},
		{
			name:   "zero",
			fields: fields{0, 0, 0, 0},
			want:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{
				HitStats: HitStats{
					Hits:         tt.fields.Hits,
					GraceHits:    tt.fields.GraceHits,
					Misses:       tt.fields.Misses,
					Replacements: tt.fields.Replacements,
				},
			}
			assert.InDeltaf(t, tt.want, s.HitRatio(), 0.001, "HitRatio()")
		})
	}
}

func TestCache_HitStats(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			replaceFn := func(ctx context.Context, key string) (string, error) {
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 250*time.Millisecond, 500*time.Millisecond, c.cacheOpts...)
			assert.NoError(t, err)

			t0 := time.Now()
			v, err := cache.Get(context.Background(), "k1") // Miss -> Sync Replacement
			assert.NoError(t, err)
			assert.Equal(t, "result-k1", v)
			assert.Equal(t, HitStats{0, 0, 1, 1}, cache.Stats().HitStats)

			v, err = cache.Get(context.Background(), "k1") // Hit
			assert.NoError(t, err)
			assert.Equal(t, "result-k1", v)
			assert.Equal(t, HitStats{1, 0, 1, 1}, cache.Stats().HitStats)

			v, err = cache.Get(context.Background(), "k2") // Miss -> Sync Replacement
			assert.NoError(t, err)
			assert.Equal(t, "result-k2", v)
			assert.Equal(t, HitStats{1, 0, 2, 2}, cache.Stats().HitStats)

			time.Sleep(300 * time.Millisecond)
			v, err = cache.Get(context.Background(), "k1") // Grace Hit
			assert.NoError(t, err)
			assert.Equal(t, "result-k1", v)

			// Sleep for some time - background fetch causes race condition on Replacements
			time.Sleep(50 * time.Millisecond)
			assert.Equal(t, HitStats{1, 1, 2, 3}, cache.Stats().HitStats)
			// assert t=350ms
			assert.InDelta(t, 350*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}

func TestCache_SizeStats(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches(10) {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			replaceFn := func(ctx context.Context, key string) (string, error) {
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 250*time.Millisecond, 500*time.Millisecond, c.cacheOpts...)
			assert.NoError(t, err)

			stats := cache.Stats().SizeStats
			assert.Equal(t, 0, stats.Size)
			assert.True(t, stats.Capacity == -1 || stats.Capacity == 10)

			_, err = cache.Get(context.Background(), "k1")
			assert.NoError(t, err)

			stats = cache.Stats().SizeStats
			assert.Equal(t, 1, stats.Size)
			assert.True(t, stats.Capacity == -1 || stats.Capacity == 10)
		})
	}

	for _, c := range evictingCaches(10) {
		c := c
		t.Run(c.name+" (evicting check)", func(t *testing.T) {
			t.Parallel()

			replaceFn := func(ctx context.Context, key string) (string, error) {
				return "result-" + key, nil
			}
			cache, err := New[string, string](replaceFn, 250*time.Millisecond, 500*time.Millisecond, c.cacheOpts...)
			assert.NoError(t, err)

			assert.Equal(t, SizeStats{0, 10}, cache.Stats().SizeStats)

			for i := 0; i < 10; i++ {
				_, err := cache.Get(context.Background(), "k1-"+strconv.Itoa(i))
				assert.NoError(t, err)
				assert.Equal(t, SizeStats{i + 1, 10}, cache.Stats().SizeStats)
			}

			for i := 0; i < 10; i++ {
				_, err := cache.Get(context.Background(), "k2-"+strconv.Itoa(i))
				assert.NoError(t, err)
				assert.Equal(t, SizeStats{10, 10}, cache.Stats().SizeStats)
			}
		})
	}
}
