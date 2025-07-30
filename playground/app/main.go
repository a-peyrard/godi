package main

import (
	"context"
	"errors"
	"github.com/a-peyrard/godi"
	"github.com/a-peyrard/godi/playground/app/registry"
	"github.com/a-peyrard/godi/runner"
	"log"
)

func main() {
	resolver := godi.New()
	//goland:noinspection GoUnhandledErrorResult
	defer resolver.Close()

	resolver.MustRegister(func() context.Context {
		return runner.WithSyscallKillableContext(context.Background())
	})
	registry.Registry{}.Register(resolver)

	if err := runner.Run(resolver); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("Error running app: %v", err)
	}

	log.Println("bye.")
}
