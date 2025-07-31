package slices

import (
	"errors"
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

func TestMap(t *testing.T) {
	t.Run("it should transform strings to their lengths", func(t *testing.T) {
		// GIVEN
		input := []string{"foo", "bar", "hello", "augustin"}
		mapper := func(s string) int {
			return len(s)
		}

		// WHEN
		result := Map(input, mapper)

		// THEN
		assert.Equal(t, []int{3, 3, 5, 8}, result)
	})

	t.Run("it should transform integers to strings", func(t *testing.T) {
		// GIVEN
		input := []int{1, 2, 3, 4, 5}
		mapper := func(n int) string {
			return string(rune('0' + n))
		}

		// WHEN
		result := Map(input, mapper)

		// THEN
		assert.Equal(t, []string{"1", "2", "3", "4", "5"}, result)
	})

	t.Run("it should transform integers by doubling them", func(t *testing.T) {
		// GIVEN
		input := []int{1, 2, 3, 4, 5}
		mapper := func(n int) int {
			return n * 2
		}

		// WHEN
		result := Map(input, mapper)

		// THEN
		assert.Equal(t, []int{2, 4, 6, 8, 10}, result)
	})

	t.Run("it should handle empty slice", func(t *testing.T) {
		// GIVEN
		var input []string
		mapper := func(s string) int {
			return len(s)
		}

		// WHEN
		result := Map(input, mapper)

		// THEN
		assert.Empty(t, result)
	})

	t.Run("it should transform structs to their fields", func(t *testing.T) {
		// GIVEN
		type Person struct {
			Name string
			Age  int
		}
		input := []Person{
			{Name: "Alice", Age: 30},
			{Name: "Bob", Age: 25},
			{Name: "Charlie", Age: 35},
		}
		mapper := func(p Person) string {
			return p.Name
		}

		// WHEN
		result := Map(input, mapper)

		// THEN
		assert.Equal(t, []string{"Alice", "Bob", "Charlie"}, result)
	})

	t.Run("it should handle single element slice", func(t *testing.T) {
		// GIVEN
		input := []string{"test"}
		mapper := func(s string) int {
			return len(s)
		}

		// WHEN
		result := Map(input, mapper)

		// THEN
		assert.Equal(t, []int{4}, result)
	})
}

func TestUnsafeMap(t *testing.T) {
	t.Run("it should transform strings to their lengths", func(t *testing.T) {
		// GIVEN
		input := []string{"foo", "hello", "a"}
		mapper := func(s string) (int, error) {
			return len(s), nil
		}

		// WHEN
		result, err := UnsafeMap(input, mapper)

		// THEN
		assert.NoError(t, err)
		assert.Equal(t, []int{3, 5, 1}, result)
	})

	t.Run("it should return error when mapper fails", func(t *testing.T) {
		// GIVEN
		input := []string{"foo", "bar", "baz"}
		mapper := func(s string) (int, error) {
			if s == "bar" {
				return 0, errors.New("failed to process bar")
			}
			return len(s), nil
		}

		// WHEN
		result, err := UnsafeMap(input, mapper)

		// THEN
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to process bar")
	})

	t.Run("it should handle empty slice", func(t *testing.T) {
		// GIVEN
		var input []string
		mapper := func(s string) (int, error) {
			return len(s), nil
		}

		// WHEN
		result, err := UnsafeMap(input, mapper)

		// THEN
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("it should transform integers to strings", func(t *testing.T) {
		// GIVEN
		input := []int{1, 2, 3}
		mapper := func(n int) (string, error) {
			if n < 0 {
				return "", errors.New("negative numbers not allowed")
			}
			return string(rune('0' + n)), nil
		}

		// WHEN
		result, err := UnsafeMap(input, mapper)

		// THEN
		assert.NoError(t, err)
		assert.Equal(t, []string{"1", "2", "3"}, result)
	})
}
