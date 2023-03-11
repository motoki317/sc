package sc

import (
	"sync"
)

// call is an in-flight or completed cache replacement call.
type call[V any] struct {
	wg sync.WaitGroup

	// These fields are written once before the WaitGroup is done
	// and are only read after the WaitGroup is done.
	val value[V]
	err error
}
