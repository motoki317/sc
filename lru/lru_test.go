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
			c := lru.New[int, int](lru.WithCapacity(tc.capacity))
			for i := 0; i < tc.capacity+1; i++ {
				c.Set(i, i)
			}

			require.Equal(t, tc.capacity, c.Len(), "expected capacity to be full")

			_, ok := c.Get(0)
			require.False(t, ok, "expected key to be evicted")

			_, ok = c.Get(1)
			require.True(t, ok, "expected key to exist")
		})
	}
}

func TestCache_Get(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		c := lru.New[int, int]()

		_, ok := c.Get(0)

		require.False(t, ok, "expected not ok")
	})
	t.Run("existing", func(t *testing.T) {
		c := lru.New[int, int]()
		value := 100

		c.Set(1, value)
		actual, ok := c.Get(1)

		require.True(t, ok, "expected ok")
		require.Equal(t, value, actual, "expected set value to be %s", value)
	})
}

func TestCache_Peek(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		c := lru.New[int, int]()

		_, ok := c.Peek(1)

		require.False(t, ok, "expected not ok")
	})
	t.Run("existing", func(t *testing.T) {
		c := lru.New[int, int]()

		c.Set(1, 1)
		value, ok := c.Peek(1)

		require.True(t, ok, "expected ok")
		require.Equal(t, 1, value, "expected peek value to be 1")
	})
}

func TestCache_Set(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		c := lru.New[int, int]()

		c.Set(1, 1)
		value, ok := c.Get(1)

		require.True(t, ok, "expected ok")
		require.Equal(t, 1, value, "expected set value to be 1")
	})
	t.Run("existing", func(t *testing.T) {
		c := lru.New[int, int]()

		c.Set(1, 1)
		c.Set(1, 2)
		value, ok := c.Get(1)

		require.True(t, ok, "expected ok")
		require.Equal(t, 2, value, "expected set value to be2")
	})
}

func TestCache_Delete(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		c := lru.New[int, int]()

		key := 100
		c.Delete(key)

		_, ok := c.Get(key)
		require.False(t, ok, "expected not ok")
	})
	t.Run("existing", func(t *testing.T) {
		c := lru.New[int, int]()

		key, value := 1, 100
		c.Set(key, value)
		require.Equal(t, 1, c.Len())

		c.Delete(key)

		_, ok := c.Get(key)
		require.False(t, ok, "expected not ok")
		require.Equal(t, 0, c.Len())
	})
}

func TestCache_DeleteIf(t *testing.T) {
	c := lru.New[int, int]()

	c.Set(1, 10)
	c.Set(2, 10)
	c.Set(3, 10)
	c.Set(4, 10)

	c.DeleteIf(func(key int, value int) bool {
		return key%2 == 0
	})

	require.Equal(t, 2, c.Len())
	_, ok := c.Peek(1)
	require.True(t, ok)
	_, ok = c.Peek(2)
	require.False(t, ok)
	_, ok = c.Peek(3)
	require.True(t, ok)
	_, ok = c.Peek(4)
	require.False(t, ok)
}

func TestCache_DeleteOldest(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		c := lru.New[int, int]()

		_, _, ok := c.DeleteOldest()

		require.False(t, ok, "expected not ok")
	})
	t.Run("existing", func(t *testing.T) {
		c := lru.New[int, int]()

		c.Set(1, 10)
		c.Set(2, 20)
		c.Set(3, 30)

		_, _ = c.Get(1)
		_, _ = c.Get(2)
		_, _ = c.Get(3)

		key, value, ok := c.DeleteOldest()

		require.True(t, ok, "expected ok")
		require.Equal(t, 1, key, "expected key to be 1")
		require.Equal(t, 10, value, "expected value to be 10")
	})
}

func TestCache_Purge(t *testing.T) {
	c := lru.New[int, int]()

	key, value := 1, 100
	c.Set(key, value)
	require.Equal(t, c.Len(), 1)

	c.Purge()
	require.Equal(t, c.Len(), 0)

	_, ok := c.Get(key)
	require.False(t, ok, "expected not ok")
}
