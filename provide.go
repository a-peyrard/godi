package godi

import (
	"fmt"
	"reflect"
)

type (
	dependencyVertex struct {
		name         Name
		resolved     *reflect.Value
		provider     Provider
		dependencies []dependencyVertex
	}
)

func (r *Resolver) provideUsing(p Provider, name Name) (reflect.Value, error) {
	// here we need to compute the dependency graph
	dependencies := make([]reflect.Value, len(p.Dependencies()))
	for idx, depReq := range p.Dependencies() {
		val, _, err := r.resolve(depReq)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to resolve dependency %v to provide component %s:\n\t%w", depReq, name, err)
		}
		dependencies[idx] = val
	}

	comp, err := p.Provide(name, dependencies)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("failed to provide component %s using provider %s:\n\t%w", name, p, err)
	}

	// store the component in the store for future use
	r.store.Put(name, comp)

	// fixme: handle cycles!!!

	return comp, nil
}
