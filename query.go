package godi

import (
	"fmt"
	"reflect"
)

type (
	query interface {
		find(r *Resolver) ([]*providerDef, error)

		fmt.Stringer
	}

	queryByType struct {
		typ reflect.Type
	}

	queryByName struct {
		name Name
	}
)

func (q queryByType) find(r *Resolver) ([]*providerDef, error) {
	basket := map[Name]*providerDef{}
	for name, providers := range r.providers {
		if matchType(q.typ, name.typ) && providers.IsNotEmpty() {
			basket[name] = providers.Peek()
		}
	}

	// look what we have in stock for dynamic providers
	registeredNewOnes := false
	for _, dynamicP := range r.dynamicProviders {
		for _, n := range dynamicP.ListBuildableNames() {
			if _, found := basket[n]; !found && matchType(q.typ, n.typ) {
				provider, opts, err := dynamicP.BuildProviderFor(n)
				if err != nil {
					return nil, fmt.Errorf("failed to build provider for %s:\n\t%w", n, err)
				}
				err = r.Register(provider, opts...)
				if err != nil {
					return nil, fmt.Errorf("failed to register built provider for %s:\n\t%w", n, err)
				}
				registeredNewOnes = true
			}
		}
	}

	if registeredNewOnes {
		// Re-query after dynamic providers have been registered.
		//
		// We cannot use the registered provider as soon as it is registered, because multiple
		// providers might have been generated, with different priorities, ... so we need to re-query
		// what we have in stock.
		return q.find(r)
	}

	values := make([]*providerDef, 0, len(basket))
	for _, v := range basket {
		values = append(values, v)
	}
	return values, nil
}

func (q queryByType) String() string {
	return fmt.Sprintf("<type ~= %s>", q.typ.String())
}

func (q queryByName) find(r *Resolver) ([]*providerDef, error) {
	var basket []*providerDef
	for name, providers := range r.providers {
		if matchType(q.name.typ, name.typ) && q.name.name == name.name && providers.IsNotEmpty() {
			basket = append(basket, providers.Peek())
		}
	}

	// look for dynamic providers if we didn't find anything yet
	if len(basket) == 0 {
		registeredAtLeastOne := false
		for _, dynamicP := range r.dynamicProviders {
			if dynamicP.CanBuild(q.name) {
				provider, opts, err := dynamicP.BuildProviderFor(q.name)
				if err != nil {
					return nil, fmt.Errorf("failed to build provider for %s:\n\t%w", q.name, err)
				}
				err = r.Register(provider, opts...)
				if err != nil {
					return nil, fmt.Errorf("failed to register built provider for %s:\n\t%w", q.name, err)
				}

				registeredAtLeastOne = true
			}
		}

		if registeredAtLeastOne {
			// Re-query after dynamic providers have been registered.
			//
			// We cannot use the registered provider as soon as it is registered, because multiple
			// providers might have been generated, with different priorities, ... so we need to re-query
			// what we have in stock.
			return q.find(r)
		}
	}

	return basket, nil
}

func (q queryByName) findInProvidersOnly(r *Resolver) []*providerDef {
	var basket []*providerDef
	for name, providers := range r.providers {
		if matchType(q.name.typ, name.typ) && q.name.name == name.name && providers.IsNotEmpty() {
			basket = append(basket, providers.Peek())
		}
	}
	return basket
}

func (q queryByName) String() string {
	return fmt.Sprintf("<type ~= %s and name = %s>", q.name.typ.String(), q.name.name)
}

func matchType(queryType, providedType reflect.Type) bool {
	if queryType == providedType {
		return true
	}
	if queryType.Kind() == reflect.Interface && providedType.Implements(queryType) {
		return true
	}
	return false
}
