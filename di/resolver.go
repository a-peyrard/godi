package main

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"os"
	"reflect"
	"strings"
	"time"
)

type (
	Provider any

	providerDef struct {
		provides     reflect.Type
		fn           reflect.Value
		dependencies []reflect.Type
		instance     *reflect.Value
	}

	Resolver struct {
		providers map[reflect.Type]*providerDef
	}
)

func New() *Resolver {
	return &Resolver{
		providers: make(map[reflect.Type]*providerDef),
	}
}

func (r *Resolver) Register(provider Provider) error {
	t := reflect.TypeOf(provider)
	if t.Kind() != reflect.Func {
		return errors.New("provider must be a function")
	}

	if t.NumOut() != 2 {
		return errors.New("provider must return two values: an instance and an error")
	}
	if t.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return errors.New("provider must return an error as the second value")
	}
	provides := t.Out(0)
	if _, exists := r.providers[provides]; exists {
		return fmt.Errorf("provider already registered for type: %s", provides.String())
	}

	paramTypes := make([]reflect.Type, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		paramType := t.In(i)
		paramTypes[i] = paramType
	}

	r.providers[provides] = &providerDef{
		provides:     provides,
		fn:           reflect.ValueOf(provider),
		dependencies: paramTypes,
	}

	return nil
}

func Resolve[T any](resolver *Resolver) (T, error) {
	var zero T
	lookFor := reflect.TypeOf(zero)
	if lookFor == nil {
		return zero, fmt.Errorf("type %T is not a valid type", zero)
	}

	provider, err := resolver.resolve(lookFor)
	if err != nil {
		return zero, fmt.Errorf("failed to resolve type %s: %w", lookFor.String(), err)
	}
	typedProvider, ok := provider.Interface().(T)
	if !ok {
		return zero, fmt.Errorf("resolved provider is not of type %s", lookFor.String())
	}

	return typedProvider, nil
}

func (r *Resolver) resolve(lookFor reflect.Type) (reflect.Value, error) {
	provider, exists := r.providers[lookFor]
	if !exists {
		return reflect.Value{}, fmt.Errorf("provider for type %s not registered", lookFor.String())
	}
	var instance reflect.Value
	if provider.instance == nil {
		var err error
		instance, err = r.generateInstance(lookFor)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to generate instance for type %s: %w", lookFor.String(), err)
		}
		provider.instance = &instance
	} else {
		instance = *provider.instance
	}

	return instance, nil
}

func (r *Resolver) generateInstance(lookFor reflect.Type) (reflect.Value, error) {
	def, exists := r.providers[lookFor]
	if !exists {
		return reflect.Value{}, fmt.Errorf("no provider registered for type %s", lookFor.String())
	}

	fmt.Printf("Resolving %s, need dependencies: %v\n", lookFor.String(), def.dependencies)
	dependencies := make([]reflect.Value, len(def.dependencies))
	for i, depType := range def.dependencies {
		dep, err := r.resolve(depType)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to resolve dependency %s for provider %s: %w", depType.String(), def.provides.String(), err)
		}
		if !dep.IsValid() {
			return reflect.Value{}, fmt.Errorf("resolved dependency %s is invalid for provider %s", depType.String(), def.provides.String())
		}
		dependencies[i] = dep
	}

	// panic recovery, as `Call` can panic if the provider function has a panic
	var results []reflect.Value
	var callErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				callErr = fmt.Errorf("panic calling provider for %s: %v", lookFor.String(), r)
			}
		}()
		results = def.fn.Call(dependencies)
	}()

	if callErr != nil {
		return reflect.Value{}, callErr
	}

	if !results[1].IsNil() {
		return reflect.Value{}, results[1].Interface().(error)
	}

	return results[0], nil
}

// -------------------------------------- PLAYGROUND CODE --------------------------------------

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

type App struct {
	Logger *zerolog.Logger
	Foobar *Foobar
}

func NewApp(foobar *Foobar, logger *zerolog.Logger) (*App, error) {
	return &App{
		Foobar: foobar,
		Logger: logger,
	}, nil
}

func (a *App) Run() {
	a.Logger.Info().Msgf("Running app with Foobar: %s", a.Foobar.Name)
}

func main() {
	// should be done in modules, each module registers its own providers
	resolver := New()

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

	// RUN THE APP
	app, err := Resolve[*App](resolver)
	if err != nil {
		fmt.Printf("Error resolving App: %v\n", err)
		return
	}

	// Run the app
	app.Run()
}
