package runner

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockRunnable is a test implementation of Runnable
type mockRunnable struct {
	counter *int32
	value   int32
	err     error
	delay   time.Duration
}

func (m *mockRunnable) Run(ctx context.Context) error {
	// Increment counter immediately when run starts (before any delay or cancellation)
	if m.counter != nil {
		atomic.AddInt32(m.counter, m.value)
	}

	// Then handle delay and context cancellation
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return m.err
}

func TestRunAll(t *testing.T) {
	t.Run("it should run all runnables successfully", func(t *testing.T) {
		// GIVEN
		var counter int32
		runnable1 := &mockRunnable{counter: &counter, value: 1}
		runnable2 := &mockRunnable{counter: &counter, value: 2}
		runnable3 := &mockRunnable{counter: &counter, value: 3}

		// WHEN
		err := RunAll(context.Background(), runnable1, runnable2, runnable3)

		// THEN
		assert.NoError(t, err)
		assert.Equal(t, int32(6), atomic.LoadInt32(&counter))
	})

	t.Run("it should return error when one runnable fails", func(t *testing.T) {
		// GIVEN
		var counter int32
		runnable1 := &mockRunnable{counter: &counter, value: 1}
		runnable2 := &mockRunnable{err: errors.New("something went wrong")}
		runnable3 := &mockRunnable{counter: &counter, value: 3}

		// WHEN
		err := RunAll(context.Background(), runnable1, runnable2, runnable3)

		// THEN
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "something went wrong")
	})

	t.Run("it should handle empty runnable list", func(t *testing.T) {
		// GIVEN / WHEN
		err := RunAll(context.Background())

		// THEN
		assert.NoError(t, err)
	})

	t.Run("it should respect context cancellation", func(t *testing.T) {
		// GIVEN
		ctx, cancel := context.WithCancel(context.Background())
		var started int32

		runnable1 := &mockRunnable{counter: &started, value: 1, delay: 100 * time.Millisecond}
		runnable2 := &mockRunnable{counter: &started, value: 1, delay: 100 * time.Millisecond}

		// WHEN
		go func() {
			// Wait a bit longer to ensure runnables have started
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		err := RunAll(ctx, runnable1, runnable2)

		// THEN
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
		// Both runnables should have started (incremented counter) before being cancelled
		assert.Equal(t, int32(2), atomic.LoadInt32(&started))
	})

	t.Run("it should run runnables concurrently", func(t *testing.T) {
		// GIVEN
		start := time.Now()
		duration := 50 * time.Millisecond

		runnable1 := &mockRunnable{delay: duration}
		runnable2 := &mockRunnable{delay: duration}
		runnable3 := &mockRunnable{delay: duration}

		// WHEN
		err := RunAll(context.Background(), runnable1, runnable2, runnable3)

		// THEN
		elapsed := time.Since(start)
		assert.NoError(t, err)
		// Should take roughly 50ms (concurrent) not 150ms (sequential)
		assert.Less(t, elapsed, 100*time.Millisecond, "Runnables should run concurrently")
	})
}
