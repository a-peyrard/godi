package set

// Set represents a generic set data structure
type Set[T comparable] map[T]struct{}

// New creates a new empty set
func New[T comparable]() Set[T] {
	return make(Set[T])
}

// NewWithValues creates a new set with the given values
func NewWithValues[T comparable](values ...T) Set[T] {
	s := New[T]()
	for _, v := range values {
		s.Add(v)
	}
	return s
}

// NewFromSlice creates a new set from the given slice
func NewFromSlice[T comparable](slice []T) Set[T] {
	var s Set[T] = make(map[T]struct{}, len(slice))
	for _, elem := range slice {
		s.Add(elem)
	}
	return s
}

// Add adds a value to the set
func (s Set[T]) Add(value T) {
	s[value] = struct{}{}
}

// Contains checks if a value exists in the set
func (s Set[T]) Contains(value T) bool {
	_, exists := s[value]
	return exists
}

// DoesNotContain checks if a value does not exist in the set
func (s Set[T]) DoesNotContain(value T) bool {
	return !s.Contains(value)
}

// Remove removes a value from the set
func (s Set[T]) Remove(value T) {
	delete(s, value)
}

// Size returns the number of elements in the set
func (s Set[T]) Size() int {
	return len(s)
}

// IsEmpty returns true if the set is empty
func (s Set[T]) IsEmpty() bool {
	return len(s) == 0
}

// ToSlice returns all values as a slice
func (s Set[T]) ToSlice() []T {
	result := make([]T, 0, len(s))
	for value := range s {
		result = append(result, value)
	}
	return result
}

// Clear removes all elements from the set
func (s Set[T]) Clear() {
	for k := range s {
		delete(s, k)
	}
}

// Union returns a new set containing all elements from both sets
func (s Set[T]) Union(other Set[T]) Set[T] {
	result := New[T]()
	for value := range s {
		result.Add(value)
	}
	for value := range other {
		result.Add(value)
	}
	return result
}

// Intersection returns a new set containing only elements present in both sets
func (s Set[T]) Intersection(other Set[T]) Set[T] {
	result := New[T]()
	for value := range s {
		if other.Contains(value) {
			result.Add(value)
		}
	}
	return result
}

// Difference returns a new set containing elements in s but not in other
func (s Set[T]) Difference(other Set[T]) Set[T] {
	result := New[T]()
	for value := range s {
		if !other.Contains(value) {
			result.Add(value)
		}
	}
	return result
}
