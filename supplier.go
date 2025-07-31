package godi

func ToStaticProvider[T any](value T) func() T {
	return func() T {
		return value
	}
}
