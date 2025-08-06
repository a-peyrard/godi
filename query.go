package godi

import (
	"fmt"
	"reflect"
)

type (
	query interface {
		find(r *Resolver) ([]*queryResult, error)

		fmt.Stringer
	}

	queryResult struct {
		name      Name
		component *reflect.Value
		provider  Provider
	}

	queryByType struct {
		typ reflect.Type
	}

	queryByName struct {
		name Name
	}
)

func (q queryByType) find(r *Resolver) ([]*queryResult, error) {
	// find all the providable names that match the type
	nameWithProviderMap := make(map[Name]*queryResult)
	for _, provider := range r.providers.All() {
		namesForProvider := provider.ListProvidableNames()
		for _, n := range namesForProvider {
			if _, exists := nameWithProviderMap[n]; !exists && matchType(q.typ, n.typ) {
				var comp *reflect.Value = nil
				if storedComp, found := r.store.Get(n); found {
					comp = &storedComp
				}
				nameWithProviderMap[n] = &queryResult{
					name:      n,
					component: comp,
					provider:  provider,
				}
			}
		}
	}

	values := make([]*queryResult, 0, len(nameWithProviderMap))
	for _, v := range nameWithProviderMap {
		values = append(values, v)
	}
	return values, nil
}

func (q queryByType) String() string {
	return fmt.Sprintf("<type~=%s>", q.typ.String())
}

func (q queryByName) find(r *Resolver) ([]*queryResult, error) {
	comp, found := r.store.Get(q.name)
	if found {
		return []*queryResult{
			{
				name:      q.name,
				component: &comp,
				provider:  nil,
			},
		}, nil
	}

	for _, provider := range r.providers.All() {
		if provider.CanProvide(q.name) {
			return []*queryResult{
				{
					name:      q.name,
					component: nil,
					provider:  provider,
				},
			}, nil
		}
	}

	return []*queryResult{}, nil
}

func (q queryByName) String() string {
	return fmt.Sprintf("<type~=%s & name=%s>", q.name.typ.String(), q.name.name)
}
