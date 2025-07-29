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

// UnsafeMap maps values of a slice using a specified mapper that can return an error.
// If the mapper returns an error, this method will return the error and stop processing the slice.
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
