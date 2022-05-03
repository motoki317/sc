package internal_test

import (
	"testing"

	"github.com/motoki317/sc/lru/internal"

	"github.com/stretchr/testify/require"
)

func TestElement_Next(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		ll := internal.NewList[int]()

		e := ll.PushFront(1)

		require.Nil(t, e.Next())
	})
	t.Run("next", func(t *testing.T) {
		ll := internal.NewList[int]()

		e1 := ll.PushFront(1)
		e2 := ll.PushFront(2)

		require.Equal(t, e1, e2.Next())
	})
}

func TestElement_Prev(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		ll := internal.NewList[int]()

		e := ll.PushFront(1)

		require.Nil(t, e.Prev())
	})
	t.Run("next", func(t *testing.T) {
		ll := internal.NewList[int]()

		e1 := ll.PushFront(1)
		e2 := ll.PushFront(2)

		require.Equal(t, e2, e1.Prev())
	})
}

func TestList_PushRemove(t *testing.T) {
	ll := internal.NewList[int]()
	length := 10

	for i := 1; i <= length; i++ {
		ll.PushFront(i)
		require.Equal(t, ll.Len(), i)
	}

	for i := length; i >= 1; i-- {
		ll.Remove(ll.Back())
		require.Equal(t, ll.Len(), i-1)
	}
}

func TestList_MoveToFront(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		ll := internal.NewList[int]()
		e := ll.PushFront(0)
		require.Equal(t, e, ll.Back())

		ll.MoveToFront(e)
		require.Equal(t, e, ll.Back())
	})
	t.Run("multiple", func(t *testing.T) {
		ll := internal.NewList[int]()
		e := ll.PushFront(0)
		ll.PushFront(1)
		require.Equal(t, e, ll.Back())

		ll.MoveToFront(e)
		require.NotEqual(t, e, ll.Back())
	})
}

func TestList_Back(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		ll := internal.NewList[int]()
		e := ll.Back()
		require.Nil(t, e)
	})
	t.Run("not empty", func(t *testing.T) {
		ll := internal.NewList[int]()
		e := ll.PushFront(1)
		ll.PushFront(2)
		require.Equal(t, e, ll.Back())
	})
}

func TestInit(t *testing.T) {
	ll := internal.NewList[int]()

	ll.PushFront(1)
	require.Equal(t, ll.Len(), 1)

	ll.Init()
	require.Equal(t, ll.Len(), 0)
}
