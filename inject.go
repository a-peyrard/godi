package godi

import (
	"fmt"
	"reflect"
)

// Inject is used as a namespace for dependency injection builders.
var Inject = &injectBuilder{}

type (
	dependency interface {
		build(targetTyp reflect.Type) (Request, error)
	}

	// entry points for builders
	injectBuilder struct{}
)

type namedDependencyBuilder struct {
	named string
}

func (i *injectBuilder) Named(name string) dependency {
	return namedDependencyBuilder{named: name}
}

func (n namedDependencyBuilder) build(targetTyp reflect.Type) (Request, error) {
	return Request{
		unitaryTyp: targetTyp,
		query: queryByName{
			name: Name{name: n.named, typ: targetTyp},
		},
		validator: validatorUniqueMandatory{},
		collector: collectorUnique{},
	}, nil
}

type autoDependencyBuilder struct{}

func (i *injectBuilder) Auto() dependency {
	return autoDependencyBuilder{}
}

func (a autoDependencyBuilder) build(targetTyp reflect.Type) (Request, error) {
	return Request{
		unitaryTyp: targetTyp,
		query: queryByType{
			typ: targetTyp,
		},
		validator: validatorUniqueMandatory{},
		collector: collectorUnique{},
	}, nil
}

type multipleDependencyBuilder struct{}

func (i *injectBuilder) Multiple() dependency {
	return multipleDependencyBuilder{}
}

func (m multipleDependencyBuilder) build(targetTyp reflect.Type) (r Request, err error) {
	if targetTyp.Kind() == reflect.Slice {
		elemTyp := targetTyp.Elem()
		return Request{
			unitaryTyp: elemTyp,
			query: queryByType{
				typ: elemTyp,
			},
			validator: validatorMultiple{},
			collector: collectorMultipleAsSlice{},
		}, nil
	}
	if targetTyp.Kind() == reflect.Map {
		valueTyp := targetTyp.Elem()
		return Request{
			unitaryTyp: valueTyp,
			query: queryByType{
				typ: valueTyp,
			},
			validator: validatorMultiple{},
			collector: collectorMultipleAsMap{},
		}, nil
	}
	return r, fmt.Errorf("multiple dependencies can only be used with slice or map types, got %s", targetTyp)
}

func defaultDependencyBuilder() dependency {
	return autoDependencyBuilder{}
}
