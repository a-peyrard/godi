package hello

import (
	"context"
	"fmt"
	"github.com/a-peyrard/godi/runner"
	"log"
	"time"
)

const sleepDuration = 2 * time.Second

// NewHelloRunner creates a new Runnable that prints "Hello world" and sleeps for a specified duration.
//
// @provider named="hello.runner"
func NewHelloRunner() runner.Runnable {
	return runner.RunnableFunc(HelloRunner)
}

// NewDecorateHelloRunner creates a new Runnable that prints "Hello world" and sleeps for a specified duration.
//
// @provider named="hello.runner" priority=100
func NewDecorateHelloRunner(
	helloRunnable runner.Runnable, // @inject named="hello.runner"
) runner.Runnable {
	return runner.RunnableFunc(func(ctx context.Context) error {
		log.Println("Decorating HelloRunner")
		err := helloRunnable.Run(ctx)
		if err != nil {
			log.Printf("HelloRunner failed: %v", err)
			return err
		}
		log.Println("HelloRunner completed successfully")
		return nil
	})
}

//goland:noinspection GoNameStartsWithPackageName
func HelloRunner(ctx context.Context) error {
	log.Println("Hello world")
	log.Printf("sleeping for %s ", sleepDuration)
	for i := 0; i < int(sleepDuration.Seconds()); i++ {
		select {
		case <-ctx.Done():
			log.Println("context cancelled, exiting early")
			return ctx.Err()
		case <-time.After(time.Second):
			fmt.Print(".")
		}
	}
	fmt.Print("\n")
	log.Println("Done sleeping, exiting now.")

	return nil
}
