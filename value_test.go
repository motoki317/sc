package sc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_value_isFresh(t *testing.T) {
	type args struct {
		now      monoTime
		freshFor time.Duration
	}
	tests := []struct {
		name    string
		created monoTime
		args    args
		want    bool
	}{
		{"not fresh", monoTime(-10 * time.Minute), args{0, 5 * time.Minute}, false},
		{"exact fresh", monoTime(-5 * time.Minute), args{0, 5 * time.Minute}, true},
		{"fresh", monoTime(-3 * time.Minute), args{0, 5 * time.Minute}, true},
		{"fresh (future)", monoTime(3 * time.Minute), args{0, 5 * time.Minute}, true},
		{"fresh (distant future)", monoTime(30 * time.Minute), args{0, 5 * time.Minute}, true},
		{"fresh (now)", 0, args{0, 5 * time.Minute}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &value[string]{
				v:       "",
				created: tt.created,
			}
			assert.Equalf(t, tt.want, v.isFresh(tt.args.now, tt.args.freshFor), "isFresh(%v, %v)", tt.args.now, tt.args.freshFor)
		})
	}
}

func Test_value_isExpired(t *testing.T) {
	type args struct {
		now monoTime
		ttl time.Duration
	}
	tests := []struct {
		name    string
		created monoTime
		args    args
		want    bool
	}{
		{"expired", monoTime(-10 * time.Minute), args{0, 5 * time.Minute}, true},
		{"exact not expired", monoTime(-5 * time.Minute), args{0, 5 * time.Minute}, false},
		{"not expired", monoTime(-3 * time.Minute), args{0, 5 * time.Minute}, false},
		{"not expired (future)", monoTime(3 * time.Minute), args{0, 5 * time.Minute}, false},
		{"not expired (distant future)", monoTime(30 * time.Minute), args{0, 5 * time.Minute}, false},
		{"not expired (now)", 0, args{0, 5 * time.Minute}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &value[string]{
				v:       "",
				created: tt.created,
			}
			assert.Equalf(t, tt.want, v.isExpired(tt.args.now, tt.args.ttl), "isExpired(%v, %v)", tt.args.now, tt.args.ttl)
		})
	}
}
