package godi

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"

	"github.com/a-peyrard/godi/option"
)

type (
	FactoryMethodProvider struct {
		name         Name
		factory      reflect.Value
		dependencies []Request

		priority int
	}
)

func NewFactoryMethodProvider(
	factoryMethod any,
	opts ...option.Option[RegistrableOptions],
) (Provider, error) {
	t := reflect.TypeOf(factoryMethod)
	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("factory method must be a function")
	}
	if t.NumOut() != 1 && t.NumOut() != 2 {
		return nil, errors.New("factory method must either return the instance and an error, or just the instance")
	}
	if t.NumOut() == 2 {
		if t.Out(1) != ErrorType {
			return nil, errors.New("if factory method returns two elements, it must return an error as the second element")
		}
	}

	fnName := runtime.FuncForPC(reflect.ValueOf(factoryMethod).Pointer()).Name()
	options := option.Build(
		&RegistrableOptions{
			named:    filepath.Base(fnName),
			priority: 0,
		},
		opts...,
	)

	var (
		provides     = t.Out(0)
		paramQueries = make([]Request, t.NumIn())
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
			return nil, fmt.Errorf("failed to build dependency for parameter %d of factory method %s:\n\t%w", i, fnName, err)
		}
	}

	return &FactoryMethodProvider{
		name: Name{
			name: options.named,
			typ:  provides,
		},
		factory:      reflect.ValueOf(factoryMethod),
		dependencies: paramQueries,
		priority:     options.priority,
	}, nil
}

func (f *FactoryMethodProvider) CanProvide(name Name) bool {
	return name.name == f.name.name && matchType(name.typ, f.name.typ)
}

func (f *FactoryMethodProvider) Provide(_ Name, dependencies []reflect.Value) (comp reflect.Value, err error) {
	// panic recovery, as `Call` can panic if the factory method has a panic
	var results []reflect.Value
	var callErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				callErr = fmt.Errorf("panic calling provider for %s: %v", f.name.String(), r)
			}
		}()
		results = f.factory.Call(dependencies)
	}()

	if callErr != nil {
		return reflect.Value{}, callErr
	}

	if len(results) == 2 && !results[1].IsNil() {
		return reflect.Value{}, results[1].Interface().(error)
	}

	return results[0], nil
}

func (f *FactoryMethodProvider) Dependencies() []Request {
	return f.dependencies
}

func (f *FactoryMethodProvider) ListProvidableNames() []Name {
	return []Name{f.name}
}

func (f *FactoryMethodProvider) Priority() int {
	return f.priority
}
