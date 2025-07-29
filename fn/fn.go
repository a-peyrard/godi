package fn

// ComparisonResult represents the result of comparing two values.
type ComparisonResult int

const (
	Equal   ComparisonResult = 0
	Less    ComparisonResult = -1
	Greater ComparisonResult = 1
)

// Comparator represents a function that compares two values of type T.
type Comparator[T any] func(i1 T, i2 T) ComparisonResult

// ReverseComparator returns a comparator that reverses the order of the given comparator.
func ReverseComparator[T any](comparator Comparator[T]) Comparator[T] {
	return func(i1 T, i2 T) ComparisonResult {
		return comparator(i2, i1)
	}
}
