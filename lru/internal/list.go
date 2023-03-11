package internal

// Element is an element in a linked list.
type Element[T any] struct {
	prev, next *Element[T]

	Value T
}

// List implements a generic linked list based off of container/list. This
// contains the minimum functionally required for an LRU cache.
type List[T any] struct {
	root Element[T]
	len  int
}

// NewList creates a new linked list.
func NewList[T any]() *List[T] {
	l := &List[T]{}
	l.Init()
	return l
}

// Init initializes the list with no elements.
func (l *List[T]) Init() {
	l.root = Element[T]{}
	l.root.prev = &l.root
	l.root.next = &l.root
	l.len = 0
}

// Len is the number of elements in the list.
func (l *List[T]) Len() int {
	return l.len
}

// Next returns the next item in the list.
func (l *List[T]) Next(e *Element[T]) *Element[T] {
	if e.next == &l.root {
		return nil
	}
	return e.next
}

// Prev returns the previous item in the list.
func (l *List[T]) Prev(e *Element[T]) *Element[T] {
	if e.prev == &l.root {
		return nil
	}
	return e.prev
}

// MoveToFront moves the given element to the front of the list.
func (l *List[T]) MoveToFront(e *Element[T]) {
	if l.root.next == e { // Already at front
		return
	}

	// Remove
	e.prev.next = e.next
	e.next.prev = e.prev

	// Push front
	e.prev = &l.root
	e.next = l.root.next
	e.prev.next = e
	e.next.prev = e
}

// Remove removes the given element from the list.
func (l *List[T]) Remove(e *Element[T]) T {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	l.len--
	return e.Value
}

// PushFront adds a new value to the front of the list.
func (l *List[T]) PushFront(value T) *Element[T] {
	e := &Element[T]{Value: value}
	e.prev = &l.root
	e.next = l.root.next
	e.prev.next = e
	e.next.prev = e
	l.len++
	return e
}

// Back returns the last element in the list.
func (l *List[T]) Back() *Element[T] {
	if l.len == 0 {
		return nil
	}

	return l.root.prev
}
