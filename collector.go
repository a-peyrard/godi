package godi

import (
	"fmt"
	"reflect"
)

type (
	collector interface {
		collect(unitaryTyp reflect.Type, resolver *Resolver, results []*queryResult) (val reflect.Value, found bool, err error)

		fmt.Stringer
	}

	collectorUnique struct{}

	collectorMultipleAsSlice struct{}

	collectorMultipleAsMap struct{}
)

func (c collectorUnique) collect(_ reflect.Type, r *Resolver, results []*queryResult) (val reflect.Value, found bool, err error) {
	if len(results) == 0 {
		return reflect.Value{}, false, nil
	}

	return extractComponentFromResult(r, results[0])
}

func (c collectorUnique) String() string {
	return "<📦 unique>"
}

func (c collectorMultipleAsSlice) collect(unitaryTyp reflect.Type, r *Resolver, results []*queryResult) (val reflect.Value, found bool, err error) {
	length := len(results)
	slice := reflect.MakeSlice(reflect.SliceOf(unitaryTyp), length, length)
	for i, result := range results {
		comp, _, err := extractComponentFromResult(r, result)
		if err != nil {
			return reflect.Value{}, false, err
		}

		slice.Index(i).Set(comp)
	}

	return slice, true, nil
}

func (c collectorMultipleAsSlice) String() string {
	return "<📦 multiple as slice>"
}

func (c collectorMultipleAsMap) collect(unitaryTyp reflect.Type, r *Resolver, results []*queryResult) (val reflect.Value, found bool, err error) {
	mapValue := reflect.MakeMapWithSize(reflect.MapOf(StringType, unitaryTyp), len(results))
	for _, result := range results {
		comp, _, err := extractComponentFromResult(r, result)
		if err != nil {
			return reflect.Value{}, false, err
		}

		mapValue.SetMapIndex(reflect.ValueOf(result.name.name), comp)
	}

	return mapValue, true, nil
}

func (c collectorMultipleAsMap) String() string {
	return "<📦 multiple as map>"
}

func extractComponentFromResult(r *Resolver, result *queryResult) (comp reflect.Value, found bool, err error) {
	if result.component != nil {
		comp = *result.component
	} else {
		comp, err = r.provideUsing(result.provider, result.name)
		if err != nil {
			return reflect.Value{}, false, fmt.Errorf("failed to provide using %s:\n\t%w", result.provider, err)
		}
	}

	return comp, true, err
}
