package sc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_value_isFresh(t *testing.T) {
	now := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	type args struct {
		now      time.Time
		freshFor time.Duration
	}
	tests := []struct {
		name string
		t    time.Time
		args args
		want bool
	}{
		{"not fresh", now.Add(-10 * time.Minute), args{now, 5 * time.Minute}, false},
		{"fresh", now.Add(-3 * time.Minute), args{now, 5 * time.Minute}, true},
		{"fresh (future)", now.Add(3 * time.Minute), args{now, 5 * time.Minute}, true},
		{"fresh (distant future)", now.Add(30 * time.Minute), args{now, 5 * time.Minute}, true},
		{"fresh (now)", now, args{now, 5 * time.Minute}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &value[string]{
				v: "",
				t: tt.t,
			}
			assert.Equalf(t, tt.want, v.isFresh(tt.args.now, tt.args.freshFor), "isFresh(%v, %v)", tt.args.now, tt.args.freshFor)
		})
	}
}

func Test_value_isExpired(t *testing.T) {
	now := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	type args struct {
		now time.Time
		ttl time.Duration
	}
	tests := []struct {
		name string
		t    time.Time
		args args
		want bool
	}{
		{"expired", now.Add(-10 * time.Minute), args{now, 5 * time.Minute}, true},
		{"not expired", now.Add(-3 * time.Minute), args{now, 5 * time.Minute}, false},
		{"not expired (future)", now.Add(3 * time.Minute), args{now, 5 * time.Minute}, false},
		{"not expired (distant future)", now.Add(30 * time.Minute), args{now, 5 * time.Minute}, false},
		{"not expired (now)", now, args{now, 5 * time.Minute}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &value[string]{
				v: "",
				t: tt.t,
			}
			assert.Equalf(t, tt.want, v.isExpired(tt.args.now, tt.args.ttl), "isExpired(%v, %v)", tt.args.now, tt.args.ttl)
		})
	}
}
