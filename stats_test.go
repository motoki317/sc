package sc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStats_String(t *testing.T) {
	type fields struct {
		Hits         uint64
		GraceHits    uint64
		Misses       uint64
		Replacements uint64
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "simple",
			fields: fields{1, 2, 3, 4},
			want:   "Hits: 1, GraceHits: 2, Misses: 3, Replacements: 4, Hit Ratio: 0.500000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{
				Hits:         tt.fields.Hits,
				GraceHits:    tt.fields.GraceHits,
				Misses:       tt.fields.Misses,
				Replacements: tt.fields.Replacements,
			}
			assert.Equalf(t, tt.want, s.String(), "String()")
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
				Hits:         tt.fields.Hits,
				GraceHits:    tt.fields.GraceHits,
				Misses:       tt.fields.Misses,
				Replacements: tt.fields.Replacements,
			}
			assert.InDeltaf(t, tt.want, s.HitRatio(), 0.001, "HitRatio()")
		})
	}
}

func TestCache_Stats(t *testing.T) {
	t.Parallel()

	for _, c := range allCaches {
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
			assert.Equal(t, Stats{0, 0, 1, 1}, cache.Stats())

			v, err = cache.Get(context.Background(), "k1") // Hit
			assert.NoError(t, err)
			assert.Equal(t, "result-k1", v)
			assert.Equal(t, Stats{1, 0, 1, 1}, cache.Stats())

			v, err = cache.Get(context.Background(), "k2") // Miss -> Sync Replacement
			assert.NoError(t, err)
			assert.Equal(t, "result-k2", v)
			assert.Equal(t, Stats{1, 0, 2, 2}, cache.Stats())

			time.Sleep(300 * time.Millisecond)
			v, err = cache.Get(context.Background(), "k1") // Grace Hit
			assert.NoError(t, err)
			assert.Equal(t, "result-k1", v)

			// Sleep for some time - background fetch causes race condition on Replacements
			time.Sleep(50 * time.Millisecond)
			assert.Equal(t, Stats{1, 1, 2, 3}, cache.Stats())
			// assert t=350ms
			assert.InDelta(t, 350*time.Millisecond, time.Since(t0), float64(100*time.Millisecond))
		})
	}
}
