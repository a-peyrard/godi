package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
)

type (
	Query interface {
		Want(name Name) bool

		fmt.Stringer
	}

	Name struct {
		name         string
		providedType reflect.Type
	}

	Provider any

	providerDef struct {
		name Name

		factory      reflect.Value
		dependencies []reflect.Type

		instance *reflect.Value

		priority int
	}

	Resolver struct {
		providers map[Name][]*providerDef
	}

	queryByType struct {
		typ reflect.Type
	}
)

func (n Name) String() string {
	return fmt.Sprintf("(%s, %s)", n.name, n.providedType.String())
}

func NewQueryForType(typ reflect.Type) Query {
	return &queryByType{
		typ: typ,
	}
}

func (q *queryByType) Want(n Name) bool {
	return q.typ == n.providedType
}

func (q *queryByType) String() string {
	return fmt.Sprintf("type = %s", q.typ.String())
}

func New() *Resolver {
	return &Resolver{
		providers: make(map[Name][]*providerDef),
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

	name := Name{
		name:         filepath.Base(funcName),
		providedType: provides,
	}

	r.providers[name] = append(r.providers[name], &providerDef{
		name: name,

		factory:      reflect.ValueOf(provider),
		dependencies: paramTypes,
	})

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
	provider, err := r.findOne(query)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("failed to find provider for query %v: %w", query, err)
	}
	return r.instantiate(provider)
}

func (r *Resolver) find(query Query) ([]*providerDef, error) {
	var basket []*providerDef
	for name, providers := range r.providers {
		if query.Want(name) {
			basket = append(basket, providers...)
		}
	}
	return basket, nil
}

func (r *Resolver) findOne(query Query) (*providerDef, error) {
	basket, err := r.find(query)
	if err != nil {
		return nil, fmt.Errorf("failed to find providers for query %v: %w", query, err)
	}

	if len(basket) == 0 {
		return nil, fmt.Errorf("no provider found for query: %v", query)
	}
	if len(basket) > 1 {
		return nil, fmt.Errorf("multiple providers found for query: %v, found: %d, use a more precise query", query, len(basket))
	}

	return basket[0], nil
}

func (r *Resolver) instantiate(provider *providerDef) (reflect.Value, error) {
	var instance reflect.Value
	if provider.instance == nil {
		var err error
		instance, err = r.makeInstance(provider)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to generate instance for type %s: %w", provider.name, err)
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
	for i, depType := range def.dependencies {
		dep, err := r.resolve(NewQueryForType(depType))
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to resolve dependency %s for provider %s: %w", depType.String(), def.name.String(), err)
		}
		if !dep.IsValid() {
			return reflect.Value{}, fmt.Errorf("resolved dependency %s is invalid for provider %s", depType.String(), def.name.String())
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

	if !results[1].IsNil() {
		return reflect.Value{}, results[1].Interface().(error)
	}

	return results[0], nil
}
