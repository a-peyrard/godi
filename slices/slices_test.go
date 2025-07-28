package slices

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilter(t *testing.T) {
	t.Run("it should filter strings by length", func(t *testing.T) {
		// GIVEN
		input := []string{"foo", "bar", "hello", "augustin", "baz"}
		predicate := func(s string) bool {
			return len(s) == 3
		}

		// WHEN
		result := Filter(input, predicate)

		// THEN
		assert.Equal(t, []string{"foo", "bar", "baz"}, result)
	})

	t.Run("it should filter integers by even numbers", func(t *testing.T) {
		// GIVEN
		input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		predicate := func(n int) bool {
			return n%2 == 0
		}

		// WHEN
		result := Filter(input, predicate)

		// THEN
		assert.Equal(t, []int{2, 4, 6, 8, 10}, result)
	})

	t.Run("it should return empty slice when no elements match", func(t *testing.T) {
		// GIVEN
		input := []string{"hello", "world", "testing"}
		predicate := func(s string) bool {
			return len(s) == 1
		}

		// WHEN
		result := Filter(input, predicate)

		// THEN
		assert.Empty(t, result)
	})

	t.Run("it should return all elements when all match", func(t *testing.T) {
		// GIVEN
		input := []string{"a", "b", "c"}
		predicate := func(s string) bool {
			return len(s) == 1
		}

		// WHEN
		result := Filter(input, predicate)

		// THEN
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("it should handle empty slice", func(t *testing.T) {
		// GIVEN
		var input []string
		predicate := func(s string) bool {
			return true
		}

		// WHEN
		result := Filter(input, predicate)

		// THEN
		assert.Empty(t, result)
	})
}
