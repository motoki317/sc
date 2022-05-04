package tq

import (
	"math/rand"
	"testing"
)

func Benchmark2Q_Rand(b *testing.B) {
	l := New[int64, int64](8192)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = rand.Int63() % 32768
	}

	b.ResetTimer()

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			l.Set(trace[i], trace[i])
		} else {
			_, ok := l.Get(trace[i])
			if ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

func Benchmark2Q_Freq(b *testing.B) {
	l := New[int64, int64](8192)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = rand.Int63() % 16384
		} else {
			trace[i] = rand.Int63() % 32768
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Set(trace[i], trace[i])
	}
	var hit, miss int
	for i := 0; i < b.N; i++ {
		_, ok := l.Get(trace[i])
		if ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

func TestCache_RandomOps(t *testing.T) {
	size := 128
	l := New[int64, int64](size)

	n := 200000
	for i := 0; i < n; i++ {
		key := rand.Int63() % 512
		r := rand.Int63()
		switch r % 3 {
		case 0:
			l.Set(key, key)
		case 1:
			l.Get(key)
		case 2:
			l.Delete(key)
		}

		if l.recent.Len()+l.frequent.Len() > size {
			t.Fatalf("bad: recent: %d freq: %d",
				l.recent.Len(), l.frequent.Len())
		}
	}
}

func TestCache_Get_RecentToFrequent(t *testing.T) {
	l := New[int, int](128)

	// Touch all the entries, should be in t1
	for i := 0; i < 128; i++ {
		l.Set(i, i)
	}
	if n := l.recent.Len(); n != 128 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.frequent.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}

	// Get should upgrade to t2
	for i := 0; i < 128; i++ {
		_, ok := l.Get(i)
		if !ok {
			t.Fatalf("missing: %d", i)
		}
	}
	if n := l.recent.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.frequent.Len(); n != 128 {
		t.Fatalf("bad: %d", n)
	}

	// Get be from t2
	for i := 0; i < 128; i++ {
		_, ok := l.Get(i)
		if !ok {
			t.Fatalf("missing: %d", i)
		}
	}
	if n := l.recent.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.frequent.Len(); n != 128 {
		t.Fatalf("bad: %d", n)
	}
}

func TestCache_Add_RecentToFrequent(t *testing.T) {
	l := New[int, int](128)

	// Add initially to recent
	l.Set(1, 1)
	if n := l.recent.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.frequent.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}

	// Add should upgrade to frequent
	l.Set(1, 1)
	if n := l.recent.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.frequent.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}

	// Add should remain in frequent
	l.Set(1, 1)
	if n := l.recent.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.frequent.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}
}

func TestCache_Add_RecentEvict(t *testing.T) {
	l := New[int, int](4)

	// Add 1,2,3,4,5 -> Evict 1
	l.Set(1, 1)
	l.Set(2, 2)
	l.Set(3, 3)
	l.Set(4, 4)
	l.Set(5, 5)
	if n := l.recent.Len(); n != 4 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.recentEvict.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.frequent.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}

	// Pull in the recently evicted
	l.Set(1, 1)
	if n := l.recent.Len(); n != 3 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.recentEvict.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.frequent.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}

	// Add 6, should cause another recent evict
	l.Set(6, 6)
	if n := l.recent.Len(); n != 3 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.recentEvict.Len(); n != 2 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.frequent.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}
}

func TestCache_DeleteIf(t *testing.T) {
	l := New[int, int](128)

	for i := 1; i <= 4; i++ {
		l.Set(i, i)
	}

	l.DeleteIf(func(key int, value int) bool { return key%2 == 0 })

	if l.Len() != 2 {
		t.Fatalf("bad len: %v", l.Len())
	}
	for i := 1; i <= 4; i++ {
		_, ok := l.Get(i)
		if ok != (i%2 != 0) {
			t.Fatalf("bad ok: %v", ok)
		}
	}
}

func TestCache(t *testing.T) {
	l := New[int, int](128)

	for i := 0; i < 256; i++ {
		l.Set(i, i)
	}
	if l.Len() != 128 {
		t.Fatalf("bad len: %v", l.Len())
	}

	for i := 0; i < 128; i++ {
		_, ok := l.Get(i)
		if ok {
			t.Fatalf("should be evicted")
		}
	}
	for i := 128; i < 256; i++ {
		_, ok := l.Get(i)
		if !ok {
			t.Fatalf("should not be evicted")
		}
	}
	for i := 128; i < 192; i++ {
		l.Delete(i)
		_, ok := l.Get(i)
		if ok {
			t.Fatalf("should be deleted")
		}
	}

	l.Purge()
	if l.Len() != 0 {
		t.Fatalf("bad len: %v", l.Len())
	}
	if _, ok := l.Get(200); ok {
		t.Fatalf("should contain nothing")
	}
}
