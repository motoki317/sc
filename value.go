package sc

import (
	"time"
)

// value represents a cache item.
//
// Value can be in one of 3 states:
// 1. A value is "fresh" for the given freshFor duration.
// 2. A value is "stale" if it is neither fresh nor expired.
// 3. A value is "expired" after the given ttl duration.
type value[V any] struct {
	v V
	t time.Time // the time the value was acquired
}

func (v *value[V]) isFresh(now time.Time, freshFor time.Duration) bool {
	return now.Before(v.t.Add(freshFor))
}

func (v *value[V]) isExpired(now time.Time, ttl time.Duration) bool {
	return now.After(v.t.Add(ttl))
}
