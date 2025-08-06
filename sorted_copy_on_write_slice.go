package godi

import (
	"github.com/a-peyrard/godi/fn"
	"sort"
	"sync"
	"sync/atomic"
)

type SortedCOWSlice[T any] struct {
	data       atomic.Pointer[[]T]
	comparator fn.Comparator[T]
	mu         sync.Mutex
}

func NewSortedCOWSlice[T any](comparator fn.Comparator[T]) *SortedCOWSlice[T] {
	cowSlice := &SortedCOWSlice[T]{
		comparator: comparator,
	}
	initial := make([]T, 0)
	cowSlice.data.Store(&initial)
	return cowSlice
}

func (r *SortedCOWSlice[T]) Add(item T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current := *r.data.Load()
	pos := sort.Search(len(current), func(i int) bool {
		return r.comparator(current[i], item) != fn.Less
	})

	newSlice := make([]T, len(current)+1)
	copy(newSlice[:pos], current[:pos])
	newSlice[pos] = item
	copy(newSlice[pos+1:], current[pos:])

	r.data.Store(&newSlice)
}

func (r *SortedCOWSlice[T]) All() []T {
	return *r.data.Load()
}

func (r *SortedCOWSlice[T]) Len() int {
	return len(*r.data.Load())
}
