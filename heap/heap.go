package heap

import (
	"container/heap"
	"github.com/a-peyrard/godi/fn"
)

// innerPriorityQueue is the type that will be used by the heap package from the standard library
type innerPriorityQueue[T any] struct {
	inner      []T
	comparator fn.Comparator[T]
}

// PriorityQueue is a priority queue implementation that uses a heap.
type PriorityQueue[T any] struct {
	*innerPriorityQueue[T]
}

// New creates a new priority queue with the given comparator.
func New[T any](comparator fn.Comparator[T]) *PriorityQueue[T] {
	return &PriorityQueue[T]{
		innerPriorityQueue: &innerPriorityQueue[T]{
			inner:      make([]T, 0),
			comparator: comparator,
		},
	}
}

func (pq *PriorityQueue[T]) Push(elem T) {
	heap.Push(pq.innerPriorityQueue, elem)
}

func (pq *PriorityQueue[T]) Pop() T {
	return heap.Pop(pq.innerPriorityQueue).(T)
}

func (pq *PriorityQueue[T]) Len() int {
	return pq.innerPriorityQueue.Len()
}

func (pq *PriorityQueue[T]) IsEmpty() bool {
	return pq.innerPriorityQueue.Len() == 0
}

func (pq *PriorityQueue[T]) IsNotEmpty() bool {
	return pq.innerPriorityQueue.Len() > 0
}

func (pq *PriorityQueue[T]) Peek() T {
	return pq.innerPriorityQueue.inner[0]
}

func (pq *innerPriorityQueue[T]) Len() int { return len(pq.inner) }

func (pq *innerPriorityQueue[T]) Less(i, j int) bool {
	return pq.comparator(pq.inner[i], pq.inner[j]) == fn.Less
}

func (pq *innerPriorityQueue[T]) Swap(i, j int) {
	pq.inner[i], pq.inner[j] = pq.inner[j], pq.inner[i]
}

func (pq *innerPriorityQueue[T]) Push(x any) {
	pq.inner = append(pq.inner, x.(T))
}

func (pq *innerPriorityQueue[T]) Pop() any {
	old := pq.inner
	n := len(old)
	item := old[n-1]
	pq.inner = old[0 : n-1]
	return item
}
