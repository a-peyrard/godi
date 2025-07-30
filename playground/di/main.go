package main

import (
	"fmt"
	"github.com/a-peyrard/godi"
	"github.com/rs/zerolog"
	"io"
	"os"
	"strings"
	"time"
)

// -------------------------------------- PLAYGROUND CODE --------------------------------------
// fixme: we should remove this at some point, this is just a playground for the DI system
//  to illustrate its API and how to use it in a real application

func NewGlobalLogLevel() (zerolog.Level, error) {
	var level zerolog.Level
	levelFromEnv := os.Getenv("LOG_LEVEL")
	if levelFromEnv == "" {
		level = zerolog.InfoLevel
	} else {
		var err error
		level, err = zerolog.ParseLevel(strings.ToLower(levelFromEnv))
		if err != nil {
			return zerolog.NoLevel, fmt.Errorf("invalid log level %s: %w", levelFromEnv, err)
		}
	}
	return level, nil
}

func NewLogger(level zerolog.Level) (*zerolog.Logger, error) {
	var writer io.Writer = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	mainLogger := zerolog.New(writer).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()

	return &mainLogger, nil
}

type Foobar struct {
	Name string
}

func NewFoobar() (*Foobar, error) {
	return &Foobar{Name: "Hello world"}, nil
}

type Environment struct {
	Name string
}

func NewProdEnvironment() (*Environment, error) {
	return &Environment{Name: "Prod"}, nil
}

func NewDevEnvironment() (*Environment, error) {
	return &Environment{Name: "Dev"}, nil
}

func NewTestEnvironment() (*Environment, error) {
	return &Environment{Name: "Test"}, nil
}

type App struct {
	logger *zerolog.Logger
	foobar *Foobar
	env    *Environment
}

func NewApp(foobar *Foobar, logger *zerolog.Logger, env *Environment) (*App, error) {
	return &App{
		foobar: foobar,
		logger: logger,
		env:    env,
	}, nil
}

func (a *App) Run() {
	a.logger.Info().Msgf("[%s] Running app with Foobar: %s", a.env.Name, a.foobar.Name)
}

func main() {
	// should be done in modules, each module registers its own providers
	resolver := godi.New()

	if err := resolver.Register(NewFoobar); err != nil {
		fmt.Printf("Error registering Foobar provider: %v\n", err)
		return
	}
	if err := resolver.Register(NewGlobalLogLevel); err != nil {
		fmt.Printf("Error registering Logger provider: %v\n", err)
		return
	}
	if err := resolver.Register(NewLogger); err != nil {
		fmt.Printf("Error registering App provider: %v\n", err)
		return
	}
	if err := resolver.Register(NewApp); err != nil {
		fmt.Printf("Error registering App provider: %v\n", err)
		return
	}
	if err := resolver.Register(NewProdEnvironment, godi.Named("NameProvider")); err != nil {
		fmt.Printf("Error registering prod NameProvider provider: %v\n", err)
		return
	}

	appEnv := os.Getenv("APP_ENV")
	switch appEnv {
	case "dev":
		if err := resolver.Register(NewDevEnvironment, godi.Named("NameProvider"), godi.Priority(100)); err != nil {
			fmt.Printf("Error registering dev NameProvider provider: %v\n", err)
			return
		}
	case "test":
		if err := resolver.Register(NewTestEnvironment, godi.Named("NameProvider"), godi.Priority(100)); err != nil {
			fmt.Printf("Error registering test NameProvider provider: %v\n", err)
			return
		}

	}

	// RUN THE APP
	app, err := godi.Resolve[*App](resolver)
	if err != nil {
		fmt.Printf("Error resolving App: %v\n", err)
		return
	}

	// Run the app
	app.Run()
}
