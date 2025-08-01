package godi

import (
	"errors"
	"fmt"
	"github.com/a-peyrard/godi/fn"
	"github.com/a-peyrard/godi/heap"
	"github.com/a-peyrard/godi/option"
	"path/filepath"
	"reflect"
	"runtime"
)

type (
	Name struct {
		name string
		typ  reflect.Type
	}

	request struct {
		unitaryTyp reflect.Type
		query      query
		collector  collector
	}

	Provider any

	providerDef struct {
		name Name

		factory      reflect.Value
		dependencies []request

		instance *reflect.Value

		priority int
	}

	Resolver struct {
		providers map[Name]*heap.PriorityQueue[*providerDef]
	}

	// Closeable is an interface that can be used to close resources.
	Closeable interface {
		Close() error
	}

	RegisterOptions struct {
		named        string
		priority     int
		dependencies []dependency
	}
)

func (n Name) String() string {
	return fmt.Sprintf("(%s, %s)", n.name, n.typ.String())
}

func (r request) String() string {
	return fmt.Sprintf("{q=%s c=%s}", r.query, r.collector)
}

func Named(name string) option.Option[RegisterOptions] {
	return func(opts *RegisterOptions) {
		opts.named = name
	}
}

func Priority(priority int) option.Option[RegisterOptions] {
	return func(opts *RegisterOptions) {
		opts.priority = priority
	}
}

func Dependencies(dependencies ...dependency) option.Option[RegisterOptions] {
	return func(opts *RegisterOptions) {
		opts.dependencies = dependencies
	}
}

func New() *Resolver {
	r := &Resolver{
		providers: make(map[Name]*heap.PriorityQueue[*providerDef]),
	}
	// register itself as a static provider if provider wants to resolve the resolver to
	// dynamically be able to resolve dependencies (not using factory method parameter injection)
	r.MustRegister(ToStaticProvider(r), Named("di.resolver"))

	return r
}

func (r *Resolver) Register(provider Provider, opts ...option.Option[RegisterOptions]) error {
	t := reflect.TypeOf(provider)
	if t.Kind() != reflect.Func {
		return errors.New("provider must be a function")
	}

	if t.NumOut() != 1 && t.NumOut() != 2 {
		return errors.New("provider must either return the instance and an error, or just the instance")
	}
	if t.NumOut() == 2 {
		if t.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
			return errors.New("if provider returns two elements, it must return an error as the second element")
		}
	}

	funcName := runtime.FuncForPC(reflect.ValueOf(provider).Pointer()).Name()
	options := option.Build(
		&RegisterOptions{
			named:    filepath.Base(funcName),
			priority: 0,
		},
		opts...,
	)

	var (
		provides     = t.Out(0)
		paramQueries = make([]request, t.NumIn())
		err          error
	)
	for i := 0; i < t.NumIn(); i++ {
		paramTyp := t.In(i)
		depDef, found := tryGetAt(options.dependencies, i)
		if !found {
			depDef = defaultDependencyBuilder()
		}
		paramQueries[i], err = depDef.build(paramTyp)
		if err != nil {
			return fmt.Errorf("failed to build dependency for parameter %d of provider %s:\n\t%w", i, funcName, err)
		}
	}

	name := Name{
		name: options.named,
		typ:  provides,
	}

	providers, exists := r.providers[name]
	if !exists {
		// create a max heap (hence reversing the comparator) on priority
		providers = heap.New[*providerDef](fn.ReverseComparator(compareByPriority))
		r.providers[name] = providers
	}

	r.providers[name].Push(&providerDef{
		name: name,

		factory:      reflect.ValueOf(provider),
		dependencies: paramQueries,

		priority: options.priority,
	})

	return nil
}

func tryGetAt[T any](slice []T, index int) (val T, found bool) {
	if index < 0 || index >= len(slice) {
		return val, false
	}
	return slice[index], true
}

func (r *Resolver) MustRegister(provider Provider, opts ...option.Option[RegisterOptions]) *Resolver {
	err := r.Register(provider, opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to register provider %T: %v", provider, err))
	}
	return r
}

func (r *Resolver) Close() error {
	closeableType := reflect.TypeOf((*Closeable)(nil)).Elem()
	closeErrors := make([]error, 0)
	for _, providers := range r.providers {
		if providers.IsEmpty() {
			continue
		}
		provider := providers.Peek()
		if provider.instance != nil && provider.instance.IsValid() && provider.name.typ.Implements(closeableType) {
			out := provider.instance.MethodByName("Close").Call(nil)
			if len(out) != 1 || !out[0].IsNil() {
				closeErrors = append(
					closeErrors,
					fmt.Errorf("failed to close provider %s: %v", provider.name, out[0].Interface()),
				)
			}
		}
	}
	return errors.Join(closeErrors...)
}

// Resolve attempts to resolve a component of type T from the resolver.
func Resolve[T any](resolver *Resolver) (T, error) {
	var zero T
	lookFor := reflect.TypeOf((*T)(nil)).Elem()
	if lookFor == nil {
		return zero, fmt.Errorf("type %T is not a valid type", zero)
	}

	val, _, err := resolveTyped[T](
		resolver,
		request{
			unitaryTyp: lookFor,
			query:      queryByType{typ: lookFor},
			collector:  collectorUniqueMandatory{},
		},
	)
	return val, err
}

// ResolveNamed attempts to resolve a named component of type T from the resolver.
func ResolveNamed[T any](resolver *Resolver, name string) (T, error) {
	var zero T
	lookFor := reflect.TypeOf((*T)(nil)).Elem()
	if lookFor == nil {
		return zero, fmt.Errorf("type %T is not a valid type", zero)
	}

	val, _, err := resolveTyped[T](
		resolver,
		request{
			unitaryTyp: lookFor,
			query: queryByName{
				name: Name{name: name, typ: lookFor},
			},
			collector: collectorUniqueMandatory{},
		},
	)
	return val, err
}

// ResolveAll attempts to resolve all components of type T from the resolver.
func ResolveAll[T any](resolver *Resolver) ([]T, error) {
	lookFor := reflect.TypeOf((*T)(nil)).Elem()

	val, _, err := resolveTyped[[]T](
		resolver,
		request{
			unitaryTyp: lookFor,
			query:      queryByType{typ: lookFor},
			collector:  collectorMultipleAsSlice{},
		},
	)
	return val, err
}

// TryResolve attempts to resolve a component of type T from the resolver.
//
// It returns the resolved value, a boolean indicating if it was found, and an error if any occurred during resolution.
func TryResolve[T any](resolver *Resolver) (value T, found bool, err error) {
	var zero T
	lookFor := reflect.TypeOf((*T)(nil)).Elem()
	if lookFor == nil {
		return zero, false, fmt.Errorf("type %T is not a valid type", zero)
	}

	return resolveTyped[T](
		resolver,
		request{
			unitaryTyp: lookFor,
			query:      queryByType{typ: lookFor},
			collector:  collectorUniqueOptional{},
		},
	)
}

func resolveTyped[T any](resolver *Resolver, req request) (val T, found bool, err error) {
	resolved, found, err := resolver.resolve(req)
	if err != nil {
		return val, false, fmt.Errorf("failed to resolve request %s:\n\t%w", req, err)
	}
	if !found {
		return val, false, nil
	}
	val, err = unReflect[T](resolved)
	return val, true, err
}

func (r *Resolver) resolve(req request) (val reflect.Value, found bool, err error) {
	providers, err := r.get(req.query)
	if err != nil {
		return reflect.Value{}, false, fmt.Errorf("failed to resolve provider(s) from request %v:\n\t%w", req, err)
	}
	return req.collector.collect(req.unitaryTyp, r, providers)
}

func (r *Resolver) get(query query) ([]*providerDef, error) {
	var basket []*providerDef
	for name, providers := range r.providers {
		if query.want(name) && providers.IsNotEmpty() {
			basket = append(basket, providers.Peek())
		}
	}
	return basket, nil
}

func (r *Resolver) instantiate(provider *providerDef) (reflect.Value, error) {
	var instance reflect.Value
	if provider.instance == nil {
		var err error
		instance, err = r.makeInstance(provider)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to generate instance for type %s:\n\t%w", provider.name, err)
		}
		provider.instance = &instance
	} else {
		instance = *provider.instance
	}

	return instance, nil
}

func (r *Resolver) makeInstance(def *providerDef) (reflect.Value, error) {
	fmt.Printf("Resolving %s, need dependencies: %v\n", def.name, def.dependencies)
	dependencies := make([]reflect.Value, len(def.dependencies))
	for i, depRequest := range def.dependencies {
		dep, _, err := r.resolve(depRequest)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to resolve dependency %s for provider %s:\n\t%w", depRequest, def.name, err)
		}
		if !dep.IsValid() {
			return reflect.Value{}, fmt.Errorf("resolved dependency %s is invalid for provider %s", depRequest, def.name)
		}
		dependencies[i] = dep
	}

	// panic recovery, as `Call` can panic if the provider function has a panic
	var results []reflect.Value
	var callErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				callErr = fmt.Errorf("panic calling provider for %s: %v", def.name.String(), r)
			}
		}()
		results = def.factory.Call(dependencies)
	}()

	if callErr != nil {
		return reflect.Value{}, callErr
	}

	if len(results) == 2 && !results[1].IsNil() {
		return reflect.Value{}, results[1].Interface().(error)
	}

	return results[0], nil
}

func compareByPriority(p1, p2 *providerDef) fn.ComparisonResult {
	if p1.priority < p2.priority {
		return fn.Less
	}
	if p1.priority > p2.priority {
		return fn.Greater
	}
	return fn.Equal
}

func unReflect[T any](v reflect.Value) (res T, err error) {
	res, ok := v.Interface().(T)
	if !ok {
		return res, fmt.Errorf("value %v is not of type %T", v, res)
	}
	return res, nil
}
