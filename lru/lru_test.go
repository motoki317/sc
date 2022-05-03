package lru_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/motoki317/sc/lru"
)

func TestCapacity(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
	}{
		{"1", 1},
		{"10", 10},
		{"100", 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lru := lru.New[int, int](lru.WithCapacity(tc.capacity))
			for i := 0; i < tc.capacity+1; i++ {
				lru.Set(i, i)
			}

			require.Equal(t, tc.capacity, lru.Len(), "expected capacity to be full")

			_, ok := lru.Get(0)
			require.False(t, ok, "expected key to be evicted")

			_, ok = lru.Get(1)
			require.True(t, ok, "expected key to exist")
		})
	}
}

func TestCache_Get(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		lru := lru.New[int, int]()

		_, ok := lru.Get(0)

		require.False(t, ok, "expected not ok")
	})
	t.Run("existing", func(t *testing.T) {
		lru := lru.New[int, int]()
		value := 100

		lru.Set(1, value)
		actual, ok := lru.Get(1)

		require.True(t, ok, "expected ok")
		require.Equal(t, value, actual, "expected set value to be %s", value)
	})
}

func TestCache_Peek(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		lru := lru.New[int, int]()

		_, ok := lru.Peek(1)

		require.False(t, ok, "expected not ok")
	})
	t.Run("existing", func(t *testing.T) {
		lru := lru.New[int, int]()

		lru.Set(1, 1)
		value, ok := lru.Peek(1)

		require.True(t, ok, "expected ok")
		require.Equal(t, 1, value, "expected peek value to be 1")
	})
}

func TestCache_Set(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		lru := lru.New[int, int]()

		lru.Set(1, 1)
		value, ok := lru.Get(1)

		require.True(t, ok, "expected ok")
		require.Equal(t, 1, value, "expected set value to be 1")
	})
	t.Run("existing", func(t *testing.T) {
		lru := lru.New[int, int]()

		lru.Set(1, 1)
		lru.Set(1, 2)
		value, ok := lru.Get(1)

		require.True(t, ok, "expected ok")
		require.Equal(t, 2, value, "expected set value to be2")
	})
}

func TestCache_Delete(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		lru := lru.New[int, int]()

		key := 100
		ok := lru.Delete(key)

		require.False(t, ok, "expected not ok")
	})
	t.Run("existing", func(t *testing.T) {
		lru := lru.New[int, int]()

		key, value := 1, 100
		lru.Set(key, value)
		require.Equal(t, lru.Len(), 1)

		ok := lru.Delete(key)
		require.True(t, ok, "expected ok")
	})
}

func TestCache_DeleteIf(t *testing.T) {
	lru := lru.New[int, int]()

	lru.Set(1, 10)
	lru.Set(2, 10)
	lru.Set(3, 10)
	lru.Set(4, 10)

	lru.DeleteIf(func(key int, value int) bool {
		return key%2 == 0
	})

	require.Equal(t, 2, lru.Len())
	_, ok := lru.Peek(1)
	require.True(t, ok)
	_, ok = lru.Peek(2)
	require.False(t, ok)
	_, ok = lru.Peek(3)
	require.True(t, ok)
	_, ok = lru.Peek(4)
	require.False(t, ok)
}

func TestCache_DeleteOldest(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		lru := lru.New[int, int]()

		_, _, ok := lru.DeleteOldest()

		require.False(t, ok, "expected not ok")
	})
	t.Run("existing", func(t *testing.T) {
		lru := lru.New[int, int]()

		lru.Set(1, 10)
		lru.Set(2, 20)
		lru.Set(3, 30)

		_, _ = lru.Get(1)
		_, _ = lru.Get(2)
		_, _ = lru.Get(3)

		key, value, ok := lru.DeleteOldest()

		require.True(t, ok, "expected ok")
		require.Equal(t, 1, key, "expected key to be 1")
		require.Equal(t, 10, value, "expected value to be 10")
	})
}

func TestCache_Flush(t *testing.T) {
	lru := lru.New[int, int]()

	key, value := 1, 100
	lru.Set(key, value)
	require.Equal(t, lru.Len(), 1)

	lru.Flush()
	require.Equal(t, lru.Len(), 0)

	_, ok := lru.Get(key)
	require.False(t, ok, "expected not ok")
}
