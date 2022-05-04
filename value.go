package sc

import (
	"time"
)

var t0 = time.Now()

// monoTime represents the elapsed nanoseconds since t0.
// This utilizes monotonic clock of package time.
//
// Just like time.Duration, the maximum representation is approximately 290 years.
type monoTime int64

func monoTimeNow() monoTime {
	return monoTime(time.Now().Sub(t0))
}

// value represents a cache item.
//
// Value can be in one of 3 states:
// 1. A value is "fresh" for the given freshFor duration.
// 2. A value is "stale" if it is neither fresh nor expired.
// 3. A value is "expired" after the given ttl duration.
type value[V any] struct {
	v V
	// created is the time the function to retrieve v was called.
	// Storing created as monoTime instead of time.Time allows GC to skip the scan of values entirely if V does not
	// contain pointers.
	created monoTime
}

func (v *value[V]) isFresh(now monoTime, freshFor time.Duration) bool {
	return now < v.created+monoTime(freshFor)
}

func (v *value[V]) isExpired(now monoTime, ttl time.Duration) bool {
	return v.created+monoTime(ttl) < now
}
