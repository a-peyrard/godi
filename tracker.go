package godi

import (
	"fmt"

	"github.com/a-peyrard/godi/set"
)

type (
	Tracker struct {
		visited set.Set[Name]
		stack   []Name
	}
)

func NewTracker() *Tracker {
	return &Tracker{
		visited: set.New[Name](),
		stack:   make([]Name, 0),
	}
}

func NewTrackerFrom(other *Tracker) *Tracker {
	return &Tracker{
		visited: set.NewFromSlice(other.visited.ToSlice()),
		stack:   other.stack,
	}
}

func (tracker *Tracker) Push(n Name) error {
	if tracker.visited.Contains(n) {
		cycle := []Name{n}
		for i := len(tracker.stack) - 1; i >= 0; i-- {
			cycle = append(cycle, tracker.stack[i])
			if tracker.stack[i] == n {
				break
			}
		}

		return fmt.Errorf("cycle found:\n%s", formatCycle(cycle))
	}
	tracker.visited.Add(n)
	tracker.stack = append(tracker.stack, n)

	return nil
}

func (tracker *Tracker) Pop() Name {
	if len(tracker.stack) == 0 {
		panic("tracker: pop from empty stack")
	}
	n := tracker.stack[len(tracker.stack)-1]
	tracker.stack = tracker.stack[:len(tracker.stack)-1]
	tracker.visited.Remove(n)

	return n
}

func formatCycle(cycle []Name) string {
	str := ""
	tabs := 0
	for i := len(cycle) - 1; i >= 0; i-- {
		prefix := ""
		if i != len(cycle)-1 {
			prefix = " -> "
		}
		str += fmt.Sprintf("%s%s%s\n", generateTabs(tabs), prefix, cycle[i])
		tabs++
	}
	return str
}

func generateTabs(n int) string {
	str := ""
	for i := 0; i < n; i++ {
		str += "\t"
	}
	return str
}
