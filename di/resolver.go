package main

import (
	"errors"
	"fmt"
	"github.com/a-peyrard/godi/slices"
	"github.com/rs/zerolog"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type (
	Query interface {
		Want(name string, providedType reflect.Type) bool

		fmt.Stringer
	}

	Provider any

	providerDef struct {
		name     string
		provides reflect.Type

		factory      reflect.Value
		dependencies []reflect.Type

		instance *reflect.Value
	}

	Resolver struct {
		providers []*providerDef
	}

	queryForType struct {
		typ reflect.Type
	}
)

func NewQueryForType(typ reflect.Type) Query {
	return &queryForType{
		typ: typ,
	}
}

func (q *queryForType) Want(_ string, providedType reflect.Type) bool {
	return q.typ == providedType
}

func (q *queryForType) String() string {
	return fmt.Sprintf("type = %s", q.typ.String())
}

func New() *Resolver {
	return &Resolver{
		providers: make([]*providerDef, 0),
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
	paramTypes := make([]reflect.Type, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		paramType := t.In(i)
		paramTypes[i] = paramType
	}
	funcName := runtime.FuncForPC(reflect.ValueOf(provider).Pointer()).Name()

	r.providers = append(
		r.providers,
		&providerDef{
			name:     filepath.Base(funcName),
			provides: provides,

			factory:      reflect.ValueOf(provider),
			dependencies: paramTypes,
		},
	)

	return nil
}

func Resolve[T any](resolver *Resolver) (T, error) {
	var zero T
	lookFor := reflect.TypeOf(zero)
	if lookFor == nil {
		return zero, fmt.Errorf("type %T is not a valid type", zero)
	}

	provider, err := resolver.resolve(NewQueryForType(lookFor))
	if err != nil {
		return zero, fmt.Errorf("failed to resolve type %s: %w", lookFor.String(), err)
	}
	typedProvider, ok := provider.Interface().(T)
	if !ok {
		return zero, fmt.Errorf("resolved provider is not of type %s", lookFor.String())
	}

	return typedProvider, nil
}

func (r *Resolver) resolve(query Query) (reflect.Value, error) {
	basket := slices.Filter(r.providers, func(p *providerDef) bool {
		return query.Want(p.name, p.provides)
	})
	if len(basket) == 0 {
		return reflect.Value{}, fmt.Errorf("no provider found for query: %v", query)
	}
	if len(basket) > 1 {
		return reflect.Value{}, fmt.Errorf("multiple providers found for query: %v, found: %d, use a more precise query", query, len(basket))
	}
	provider := basket[0]
	var instance reflect.Value
	if provider.instance == nil {
		var err error
		instance, err = r.generateInstance(provider)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to generate instance for type %s: %w", provider.provides.String(), err)
		}
		provider.instance = &instance
	} else {
		instance = *provider.instance
	}

	return instance, nil
}

func (r *Resolver) generateInstance(def *providerDef) (reflect.Value, error) {
	fmt.Printf("Resolving (%s, %s), need dependencies: %v\n", def.name, def.provides.String(), def.dependencies)
	dependencies := make([]reflect.Value, len(def.dependencies))
	for i, depType := range def.dependencies {
		dep, err := r.resolve(NewQueryForType(depType))
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
				callErr = fmt.Errorf("panic calling provider for (%s, %s): %v", def.name, def.provides.String(), r)
			}
		}()
		results = def.factory.Call(dependencies)
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
