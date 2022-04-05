package sc_test

import (
	"context"
	"fmt"
	"time"

	"github.com/motoki317/sc"
)

type Person struct {
	Name string
	Age  int
}

func (p Person) String() string {
	return fmt.Sprintf("%s: %d", p.Name, p.Age)
}

func retrievePerson(_ context.Context, name string) (*Person, error) {
	// Query to database or something...
	return &Person{
		Name: name,
		Age:  25,
	}, nil
}

func Example() {
	// Production code should not ignore errors
	cache, _ := sc.New[string, *Person](retrievePerson, 1*time.Minute, 2*time.Minute, sc.WithLRUBackend(500))

	// Query the values - the cache will automatically trigger 'retrievePerson' for each key.
	a, _ := cache.Get(context.Background(), "Alice")
	b, _ := cache.Get(context.Background(), "Bob")
	fmt.Println(a) // Use the values...
	fmt.Println(b)

	// Previous results are reused
	a, _ = cache.Get(context.Background(), "Alice")
	b, _ = cache.Get(context.Background(), "Bob")
	fmt.Println(a) // Use the values...
	fmt.Println(b)

	// Output:
	// Alice: 25
	// Bob: 25
	// Alice: 25
	// Bob: 25
}
