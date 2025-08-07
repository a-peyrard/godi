package hello

import (
	"context"
	"fmt"
	"github.com/a-peyrard/godi/runner"
	"log"
	"time"
)

const sleepDuration = 2 * time.Second

// NewHelloRunner Creates a new Runnable that prints "Hello world" and sleeps for a specified duration.
//
// @provider named="hello.runner"
func NewHelloRunner(
	foo string, // @inject named="hello.foo" optional=true
) runner.Runnable {
	return runner.RunnableFunc(func(ctx context.Context) error {
		return HelloRunner(ctx, foo)
	})
}

// just to demonstrate cycle detection
//// @provider named="hello.foo"
//func FooString(
//	bar string, // @inject named="hello.bar"
//) string {
//	return "foo"
//}
//
//// @provider named="hello.bar"
//func BarString(
//	di.Runnable, // @inject named="hello.runner"
//) string {
//	return "cycle??"
//}

// OnlyDevRunner Creates a new Runnable that prints "Hello world".
//
// @provider named="hello.runner" priority=100
// @when named="APP_ENV" equals="dev"
func OnlyDevRunner() runner.Runnable {
	return runner.RunnableFunc(DevRunner)
}

//// NewDecorateHelloRunner creates a new Runnable that prints "Hello world" and sleeps for a specified duration.
////
//// @provider named="hello.runner" priority=100
//func NewDecorateHelloRunner(
//		  helloRunnable runner.Runnable, // @inject named="hello.foobar"
//        waldo string,
//        foo dispatcher.Dispatcher, // @inject named="myDispatcher"
//) runner.Runnable {
//	return runner.RunnableFunc(func(ctx context.Context) error {
//		log.Println("Decorating HelloRunner")
//		err := helloRunnable.Run(ctx)
//		if err != nil {
//			log.Printf("HelloRunner failed: %v", err)
//			return err
//		}
//		log.Println("HelloRunner completed successfully")
//		return nil
//	})
//}

//goland:noinspection GoNameStartsWithPackageName
func HelloRunner(ctx context.Context, fooBar string) error {
	log.Println("Hello world: " + fooBar)
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

func DevRunner(ctx context.Context) error {
	log.Println("Hello DEV world!")

	return nil
}
