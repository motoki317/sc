package sc_test

import (
	"context"
	"fmt"
	"time"

	"github.com/motoki317/sc"
)

type HeavyData struct {
	Data string
	// and all the gazillion fields you may have in your data
}

func retrieveHeavyData(_ context.Context, name string) (*HeavyData, error) {
	// Query to database or something...
	return &HeavyData{
		Data: "my-data-" + name,
	}, nil
}

func Example() {
	// Wrap your 'retrieveHeavyData' function with sc - it will automatically cache the values.
	// (production code should not ignore errors)
	cache, _ := sc.New[string, *HeavyData](retrieveHeavyData, 1*time.Minute, 2*time.Minute, sc.WithLRUBackend(500))

	// Query the values - the cache will automatically trigger 'retrieveHeavyData' for each key.
	foo, _ := cache.Get(context.Background(), "foo")
	bar, _ := cache.Get(context.Background(), "bar")
	fmt.Println(foo.Data) // Use the values...
	fmt.Println(bar.Data)

	// Previous results are reused, so 'retrieveHeavyData' is called only once for each key in this test.
	foo, _ = cache.Get(context.Background(), "foo")
	bar, _ = cache.Get(context.Background(), "bar")
	fmt.Println(foo.Data) // Use the values...
	fmt.Println(bar.Data)
	// Output:
	// my-data-foo
	// my-data-bar
	// my-data-foo
	// my-data-bar
}
