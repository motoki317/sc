package sc

import (
	"time"
)

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
