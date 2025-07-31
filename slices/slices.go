package slices

// Filter returns a new slice containing only the elements for which the predicate function returns true.
func Filter[T any](slice []T, predicate func(T) bool) []T {
	var result []T
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Map transforms each element of the slice using the provided mapper function.
func Map[F any, T any](original []F, mapper func(F) T) []T {
	destination := make([]T, len(original))
	for i := 0; i < len(original); i++ {
		destination[i] = mapper(original[i])
	}
	return destination
}

// UnsafeMap maps values of a slice using a specified transformer that can return an error.
// If the transformer returns an error, this method will return the error and stop processing the slice.
func UnsafeMap[F any, T any](original []F, mapper func(F) (T, error)) ([]T, error) {
	destination := make([]T, len(original))
	for i := 0; i < len(original); i++ {
		var err error
		if destination[i], err = mapper(original[i]); err != nil {
			return nil, err
		}
	}
	return destination, nil
}
