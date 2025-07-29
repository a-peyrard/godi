package runner

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Runnable represents a component that can be run with a context.
type Runnable interface {
	Run(ctx context.Context) error
}

// RunAll runs all the provided runnables concurrently and waits for all of them to finish.
//
// This method is blocking and will return an error if any of the runnables returns an error.
func RunAll(parentCtx context.Context, runnables ...Runnable) error {
	group, ctx := errgroup.WithContext(parentCtx)

	for _, runnable := range runnables {
		innerRunnable := runnable // capture loop variable
		group.Go(func() error {
			return innerRunnable.Run(ctx)
		})
	}

	return group.Wait()
}
