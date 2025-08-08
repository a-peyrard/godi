package godi

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"

	"github.com/a-peyrard/godi/option"
)

type (
	FactoryMethodDecorator struct {
		name         Name
		factory      reflect.Value
		dependencies []Request

		priority int

		description string
	}
)

func NewFactoryMethodDecorator(
	factoryMethod any,
	opts ...option.Option[RegistrableOptions],
) (Decorator, error) {
	options := option.Build(
		&RegistrableOptions{
			priority: 0,
		},
		opts...,
	)
	if options.decorate == nil {
		return nil, errors.New("no decorate option provided")
	}

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
	if t.NumIn() < 1 {
		return nil, errors.New("factory method must have at least one parameter (the component to decorate)")
	}
	if !matchType(t.In(0), t.Out(0)) {
		return nil, errors.New("the first parameter of the factory method must be the same type as the return type. Or the return type must implement the first parameter type")
	}

	fnName := runtime.FuncForPC(reflect.ValueOf(factoryMethod).Pointer()).Name()

	var (
		decorates    = t.In(0)
		paramQueries = make([]Request, t.NumIn()-1)
		err          error
	)
	for i := 0; i < t.NumIn()-1; i++ {
		paramTyp := t.In(i + 1) // first param is the component to decorate
		depDef, found := tryGetAt(options.dependencies, i)
		if !found {
			depDef = defaultDependencyBuilder()
		}
		paramQueries[i], err = depDef.build(paramTyp)
		if err != nil {
			return nil, fmt.Errorf("failed to build dependency for parameter %d of factory method %s:\n\t%w", i, fnName, err)
		}
	}

	return &FactoryMethodDecorator{
		name: Name{
			name: *options.decorate,
			typ:  decorates,
		},
		factory:      reflect.ValueOf(factoryMethod),
		dependencies: paramQueries,
		priority:     options.priority,
		description:  options.description,
	}, nil
}

func (f *FactoryMethodDecorator) ForName() Name {
	return f.name
}

func (f *FactoryMethodDecorator) Decorate(toDecorate reflect.Value, dependencies []reflect.Value) (comp reflect.Value, err error) {
	// panic recovery, as `Call` can panic if the factory method has a panic
	var results []reflect.Value
	var callErr error

	parameters := append([]reflect.Value{toDecorate}, dependencies...)
	func() {
		defer func() {
			if r := recover(); r != nil {
				callErr = fmt.Errorf("panic calling provider for %s: %v", f.name.String(), r)
			}
		}()
		results = f.factory.Call(parameters)
	}()

	if callErr != nil {
		return reflect.Value{}, callErr
	}

	if len(results) == 2 && !results[1].IsNil() {
		return reflect.Value{}, results[1].Interface().(error)
	}

	return results[0], nil
}

func (f *FactoryMethodDecorator) Dependencies() []Request {
	return f.dependencies
}

func (f *FactoryMethodDecorator) Priority() int {
	return f.priority
}

func (f *FactoryMethodDecorator) Description() string {
	return f.description
}

func (f *FactoryMethodDecorator) String() string {
	return fmt.Sprintf("FactoryMethodDecorator(%s, %s)", f.name.String(), runtime.FuncForPC(f.factory.Pointer()).Name())
}
