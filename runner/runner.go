package runner

import (
	"context"
	"fmt"
	"github.com/a-peyrard/godi"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"syscall"
)

type (
	// Runnable represents a component that can be run with a context.
	Runnable interface {
		Run(ctx context.Context) error
	}

	// RunnableFunc is a helper to create Runnable from a function.
	RunnableFunc func(ctx context.Context) error
)

func (f RunnableFunc) Run(ctx context.Context) error {
	return f(ctx)
}

// Run starts all runnables registered in the resolver with proper context handling
func Run(resolver *godi.Resolver) error {
	ctx, found, err := godi.TryResolve[context.Context](resolver)
	if err != nil {
		return fmt.Errorf("failed to resolve context: %w", err)
	}
	if !found {
		ctx = context.Background()
	}

	runnables, err := godi.ResolveAll[Runnable](resolver)
	if err != nil {
		return fmt.Errorf("failed to resolve runnables: %w", err)
	}
	if len(runnables) == 0 {
		return nil // nothing to run
	}

	return RunAll(ctx, runnables...)
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

// WithSyscallKillableContext wraps a context, and return a new context that can be canceled by system signals (SIGINT, SIGTERM, SIGKILL).
func WithSyscallKillableContext(parentCtx context.Context) context.Context {
	logger := zerolog.Ctx(parentCtx)

	ctx, cancel := context.WithCancel(parentCtx)

	go func() {
		sigterm := make(chan os.Signal, 1)
		signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

		select {
		case <-ctx.Done():
			return
		case <-sigterm:
		}
		cancel()
		logger.Info().Msg("shutting down everything...")
	}()

	return ctx
}
