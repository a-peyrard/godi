// Package option contains utility to use the variadic options pattern
package option

// Option represents a function that modifies options of type T.
type Option[T any] func(opts *T)

// Build applies a series of options to the default options struct and returns the modified result.
func Build[T any](defaultOpts *T, opts ...Option[T]) *T {
	for _, opt := range opts {
		opt(defaultOpts)
	}
	return defaultOpts
}
