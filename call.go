package sc

import (
	"sync"
)

// call is an in-flight or completed cache replacement call.
type call[V any] struct {
	wg sync.WaitGroup

	// These fields are written once before the WaitGroup is done
	// and are only read after the WaitGroup is done.
	val V
	err error

	// forgotten indicates whether Forget was called with this call's key
	// while the call was still in flight.
	forgotten bool
}
