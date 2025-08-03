package godi

import (
	"fmt"
	"reflect"
)

type (
	collector interface {
		collect(unitaryTyp reflect.Type, resolver *Resolver, providers []*providerDef) (val reflect.Value, found bool, err error)

		fmt.Stringer
	}

	collectorUniqueMandatory struct{}

	collectorUniqueOptional struct{}

	collectorMultipleAsSlice struct{}

	collectorMultipleAsMap struct{}
)

func (c collectorUniqueMandatory) collect(_ reflect.Type, r *Resolver, providers []*providerDef) (val reflect.Value, found bool, err error) {
	if len(providers) == 0 {
		return reflect.Value{}, false, fmt.Errorf("no providers found for %s", c)
	}
	if len(providers) > 1 {
		return reflect.Value{}, false, fmt.Errorf("multiple providers found for %s, expected one and only one, got %d", c, len(providers))
	}

	val, err = r.instantiate(providers[0])
	return val, true, err
}

func (c collectorUniqueMandatory) String() string {
	return "<unique mandatory>"
}

func (c collectorMultipleAsSlice) collect(unitaryTyp reflect.Type, r *Resolver, providers []*providerDef) (val reflect.Value, found bool, err error) {
	length := len(providers)
	slice := reflect.MakeSlice(reflect.SliceOf(unitaryTyp), length, length)
	for i, provider := range providers {
		instance, err := r.instantiate(provider)
		if err != nil {
			return reflect.Value{}, false, fmt.Errorf("failed to instantiate provider %s: %w", provider.name, err)
		}
		slice.Index(i).Set(instance)
	}

	return slice, true, nil
}

func (c collectorMultipleAsSlice) String() string {
	return "<multiple as slice>"
}

func (c collectorMultipleAsMap) collect(unitaryTyp reflect.Type, r *Resolver, providers []*providerDef) (val reflect.Value, found bool, err error) {
	mapValue := reflect.MakeMapWithSize(reflect.MapOf(StringType, unitaryTyp), len(providers))
	for _, provider := range providers {
		instance, err := r.instantiate(provider)
		if err != nil {
			return reflect.Value{}, false, fmt.Errorf("failed to instantiate provider %s: %w", provider.name, err)
		}
		mapValue.SetMapIndex(reflect.ValueOf(provider.name.name), instance)
	}

	return mapValue, true, nil
}

func (c collectorMultipleAsMap) String() string {
	return "<multiple as map>"
}

func (c collectorUniqueOptional) collect(_ reflect.Type, r *Resolver, providers []*providerDef) (val reflect.Value, found bool, err error) {
	if len(providers) == 0 {
		return reflect.Value{}, false, nil
	}
	if len(providers) > 1 {
		return reflect.Value{}, false, fmt.Errorf("multiple providers found for %s, expected one and only one, got %d", c, len(providers))
	}

	val, err = r.instantiate(providers[0])
	return val, true, err
}

func (c collectorUniqueOptional) String() string {
	return "<unique optional>"
}
