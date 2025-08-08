package godi

import "reflect"

type (
	Decorator interface {
		ForName() Name
		Decorate(toDecorate reflect.Value, dependencies []reflect.Value) (comp reflect.Value, err error)
		Dependencies() []Request
		Priority() int
		Description() string
	}
)
