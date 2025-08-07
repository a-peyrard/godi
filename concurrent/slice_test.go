package concurrent

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSlice_Basic(t *testing.T) {
	t.Run("it should create empty slice", func(t *testing.T) {
		// GIVEN
		slice := NewSlice[string]()

		// THEN
		assert.Equal(t, 0, slice.Length())
		assert.Empty(t, slice.Get())
	})

	t.Run("it should append elements", func(t *testing.T) {
		// GIVEN
		slice := NewSlice[string]()

		// WHEN
		slice.Append("hello")
		slice.Append("world")

		// THEN
		assert.Equal(t, 2, slice.Length())
		elements := slice.Get()
		assert.Equal(t, []string{"hello", "world"}, elements)
	})

	t.Run("it should get element at index", func(t *testing.T) {
		// GIVEN
		slice := NewSlice[string]()
		slice.Append("first")
		slice.Append("second")

		// WHEN & THEN
		assert.Equal(t, "first", slice.GetAt(0))
		assert.Equal(t, "second", slice.GetAt(1))
	})

	t.Run("it should panic on out of bounds access", func(t *testing.T) {
		// GIVEN
		slice := NewSlice[string]()
		slice.Append("only")

		// WHEN & THEN
		assert.Panics(t, func() {
			slice.GetAt(1)
		})
	})

	t.Run("it should clear all elements", func(t *testing.T) {
		// GIVEN
		slice := NewSlice[string]()
		slice.Append("hello")
		slice.Append("world")

		// WHEN
		slice.Clear()

		// THEN
		assert.Equal(t, 0, slice.Length())
		assert.Empty(t, slice.Get())
	})

	t.Run("it should return copy of slice not reference", func(t *testing.T) {
		// GIVEN
		slice := NewSlice[string]()
		slice.Append("original")

		// WHEN
		copy1 := slice.Get()
		copy2 := slice.Get()

		// THEN
		assert.NotSame(t, &copy1, &copy2, "Get() should return different slice instances")

		// Modifying the returned slice should not affect the original
		copy1[0] = "modified"
		assert.Equal(t, "original", slice.GetAt(0))
	})
}

func TestSlice_Concurrent(t *testing.T) {
	t.Run("it should handle concurrent appends", func(t *testing.T) {
		// GIVEN
		slice := NewSlice[int]()
		numGoroutines := 100
		appendsPerGoroutine := 10

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// WHEN
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < appendsPerGoroutine; j++ {
					slice.Append(goroutineID*appendsPerGoroutine + j)
				}
			}(i)
		}

		wg.Wait()

		// THEN
		assert.Equal(t, numGoroutines*appendsPerGoroutine, slice.Length())
		elements := slice.Get()
		assert.Len(t, elements, numGoroutines*appendsPerGoroutine)
	})

	t.Run("it should handle concurrent reads and writes", func(t *testing.T) {
		// GIVEN
		slice := NewSlice[string]()
		done := make(chan bool)

		// Start a writer goroutine
		go func() {
			for i := 0; i < 100; i++ {
				slice.Append("item")
				time.Sleep(time.Microsecond)
			}
			done <- true
		}()

		// Start multiple reader goroutines
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					_ = slice.Length()
					_ = slice.Get()
					time.Sleep(time.Microsecond)
				}
			}()
		}

		// WHEN
		wg.Wait()
		<-done

		// THEN
		assert.Equal(t, 100, slice.Length())
	})

	t.Run("it should handle concurrent appends and clears", func(t *testing.T) {
		// GIVEN
		slice := NewSlice[int]()
		numIterations := 50

		var wg sync.WaitGroup
		wg.Add(2)

		// Writer goroutine
		go func() {
			defer wg.Done()
			for i := 0; i < numIterations; i++ {
				slice.Append(i)
				time.Sleep(time.Microsecond)
			}
		}()

		// Clear goroutine
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				time.Sleep(5 * time.Microsecond)
				slice.Clear()
			}
		}()

		// WHEN
		wg.Wait()

		// THEN
		// The final state depends on timing, but it should not panic
		length := slice.Length()
		assert.GreaterOrEqual(t, length, 0)
		assert.LessOrEqual(t, length, numIterations)
	})

	t.Run("it should handle concurrent GetAt operations safely", func(t *testing.T) {
		// GIVEN
		slice := NewSlice[string]()

		// Pre-populate the slice
		for i := 0; i < 10; i++ {
			slice.Append("item")
		}

		var wg sync.WaitGroup
		numReaders := 20

		// WHEN
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					if slice.Length() > 0 {
						_ = slice.GetAt(0)
					}
				}
			}()
		}

		wg.Wait()

		// THEN
		assert.Equal(t, 10, slice.Length())
	})
}

func TestSlice_Types(t *testing.T) {
	t.Run("it should work with different types", func(t *testing.T) {
		// Test with int
		intSlice := NewSlice[int]()
		intSlice.Append(42)
		assert.Equal(t, 42, intSlice.GetAt(0))

		// Test with struct
		type TestStruct struct {
			Name string
			ID   int
		}
		structSlice := NewSlice[TestStruct]()
		structSlice.Append(TestStruct{Name: "test", ID: 1})
		assert.Equal(t, TestStruct{Name: "test", ID: 1}, structSlice.GetAt(0))

		// Test with pointer
		ptrSlice := NewSlice[*TestStruct]()
		testStruct := &TestStruct{Name: "ptr", ID: 2}
		ptrSlice.Append(testStruct)
		assert.Same(t, testStruct, ptrSlice.GetAt(0))
	})
}

func BenchmarkSlice_Append(b *testing.B) {
	slice := NewSlice[int]()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			slice.Append(i)
			i++
		}
	})
}

func BenchmarkSlice_Get(b *testing.B) {
	slice := NewSlice[int]()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		slice.Append(i)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = slice.Get()
		}
	})
}

func BenchmarkSlice_Length(b *testing.B) {
	slice := NewSlice[int]()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		slice.Append(i)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = slice.Length()
		}
	})
}
