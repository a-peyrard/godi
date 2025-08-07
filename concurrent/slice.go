package concurrent

import "sync"

// Slice provides a thread-safe slice implementation.
// This is primarily intended for testing purposes where thread-safe collection
// operations are needed without the complexity of channels.
type Slice[T any] struct {
	inner []T
	mu    sync.RWMutex
}

// NewSlice creates a new concurrent slice.
func NewSlice[T any]() *Slice[T] {
	return &Slice[T]{
		inner: make([]T, 0),
	}
}

// Append adds an element to the slice in a thread-safe manner.
func (s *Slice[T]) Append(v T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inner = append(s.inner, v)
}

// Get returns a copy of the current slice contents.
func (s *Slice[T]) Get() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]T, len(s.inner))
	copy(result, s.inner)
	return result
}

// GetAt returns the element at the specified index.
// Panics if index is out of bounds.
func (s *Slice[T]) GetAt(i int) T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inner[i]
}

// Length returns the current length of the slice.
func (s *Slice[T]) Length() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.inner)
}

// Clear removes all elements from the slice.
func (s *Slice[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inner = s.inner[:0]
}
