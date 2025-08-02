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

// BiConsumer represents a function that accepts two input arguments and returns no result.
type BiConsumer[T1 any, T2 any] func(t1 T1, t2 T2)

// AllBiConsumer creates a bi-consumer that will execute all the given bi-consumers.
func AllBiConsumer[A any, B any](consumers ...BiConsumer[A, B]) BiConsumer[A, B] {
	return func(a A, b B) {
		for _, consumer := range consumers {
			consumer(a, b)
		}
	}
}
