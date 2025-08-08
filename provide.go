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

func (r *Resolver) provideUsing(p Provider, name Name, tracker *Tracker) (reflect.Value, error) {
	err := tracker.Push(name)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("dependency cycle detected when trying to provide component %s using provider %s:\n\t%w", name, p, err)
	}

	lock := r.lock.GetLockFor(name)
	lock.Lock()
	defer func() {
		lock.Unlock()
		r.lock.ReleaseLock(name) // no need to store the lock anymore, we won't build the same component again
	}()

	// now that we have the lock, check if the component was built while we were waiting
	if storedComp, found := r.store.Get(name); found {
		return storedComp, nil
	}

	dependencies, err := r.resolveDependencies(p.Dependencies(), tracker)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("failed to resolve dependencies for provider %s to provide component %s:\n\t%w", p, name, err)
	}

	comp, err := p.Provide(name, dependencies)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("failed to provide component %s using provider %s:\n\t%w", name, p, err)
	}

	// check if we have decorators to apply
	decoratorsForName, found := r.decorators.Load(name)
	if found {
		for _, decorator := range decoratorsForName.(*SortedCOWSlice[Decorator]).All() {
			dependencies, err := r.resolveDependencies(decorator.Dependencies(), tracker)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("failed to resolve dependencies for decorator %s:\n\t%w", decorator, err)
			}
			comp, err = decorator.Decorate(comp, dependencies)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("failed to apply decorator %s to component %s:\n\t%w", decorator, name, err)
			}
		}
	}

	// unstack the current component from the tracker
	tracker.Pop()

	// store the component in the store for future use
	r.store.Put(name, comp)

	return comp, nil
}

func (r *Resolver) resolveDependencies(requests []Request, tracker *Tracker) ([]reflect.Value, error) {
	dependencies := make([]reflect.Value, len(requests))
	for idx, req := range requests {
		req.tracker = NewTrackerFrom(tracker)
		val, _, err := r.resolve(req)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve dependency %v:\n\t%w", req, err)
		}
		dependencies[idx] = val
	}

	return dependencies, nil
}
