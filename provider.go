package godi

import "reflect"

type (
	Provider interface {
		CanProvide(name Name) bool
		Provide(name Name, dependencies []reflect.Value) (comp reflect.Value, err error)
		Dependencies() []Request
		ListProvidableNames() []Name
		Priority() int
	}
)
