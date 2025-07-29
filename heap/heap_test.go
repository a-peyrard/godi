package heap

import (
	"testing"

	"github.com/a-peyrard/godi/fn"
	"github.com/stretchr/testify/assert"
)

type Item struct {
	Value    string
	Priority int
}

func compareByPriority(a, b *Item) fn.ComparisonResult {
	if a.Priority < b.Priority {
		return fn.Less
	}
	if a.Priority > b.Priority {
		return fn.Greater
	}
	return fn.Equal
}

func TestPriorityQueue(t *testing.T) {
	t.Run("it should create new heap", func(t *testing.T) {
		// GIVEN / WHEN
		pq := New[*Item](compareByPriority)

		// THEN
		assert.NotNil(t, pq)
		assert.True(t, pq.IsEmpty())
		assert.False(t, pq.IsNotEmpty())
		assert.Equal(t, 0, pq.Len())
	})

	t.Run("it should push and pop elements in priority order", func(t *testing.T) {
		// GIVEN
		pq := New[*Item](compareByPriority)
		item1 := &Item{Value: "low", Priority: 1}
		item2 := &Item{Value: "high", Priority: 10}
		item3 := &Item{Value: "medium", Priority: 5}

		// WHEN
		pq.Push(item1)
		pq.Push(item2)
		pq.Push(item3)

		// THEN
		assert.Equal(t, 3, pq.Len())
		assert.False(t, pq.IsEmpty())
		assert.True(t, pq.IsNotEmpty())

		// Should pop in priority order (lowest first)
		popped1 := pq.Pop()
		assert.Equal(t, "low", popped1.Value)
		assert.Equal(t, 1, popped1.Priority)

		popped2 := pq.Pop()
		assert.Equal(t, "medium", popped2.Value)
		assert.Equal(t, 5, popped2.Priority)

		popped3 := pq.Pop()
		assert.Equal(t, "high", popped3.Value)
		assert.Equal(t, 10, popped3.Priority)

		assert.True(t, pq.IsEmpty())
		assert.Equal(t, 0, pq.Len())
	})

	t.Run("it should peek without removing element", func(t *testing.T) {
		// GIVEN
		pq := New[*Item](compareByPriority)
		item1 := &Item{Value: "first", Priority: 1}
		item2 := &Item{Value: "second", Priority: 2}

		// WHEN
		pq.Push(item2)
		pq.Push(item1)

		// THEN
		peeked := pq.Peek()
		assert.Equal(t, "first", peeked.Value)
		assert.Equal(t, 2, pq.Len()) // Should not remove element

		// Verify pop still works correctly
		popped := pq.Pop()
		assert.Equal(t, "first", popped.Value)
		assert.Equal(t, 1, pq.Len())
	})

	t.Run("it should work with reverse comparator", func(t *testing.T) {
		// GIVEN
		pq := New[*Item](fn.ReverseComparator(compareByPriority))
		item1 := &Item{Value: "low", Priority: 1}
		item2 := &Item{Value: "high", Priority: 10}
		item3 := &Item{Value: "medium", Priority: 5}

		// WHEN
		pq.Push(item1)
		pq.Push(item2)
		pq.Push(item3)

		// THEN - Should pop in reverse order (highest first)
		popped1 := pq.Pop()
		assert.Equal(t, "high", popped1.Value)
		assert.Equal(t, 10, popped1.Priority)

		popped2 := pq.Pop()
		assert.Equal(t, "medium", popped2.Value)
		assert.Equal(t, 5, popped2.Priority)

		popped3 := pq.Pop()
		assert.Equal(t, "low", popped3.Value)
		assert.Equal(t, 1, popped3.Priority)
	})
}
