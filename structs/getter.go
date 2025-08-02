package structs

import (
	"fmt"
	"github.com/a-peyrard/godi/reflectutils"
	"reflect"
	"strings"
)

// Get retrieves the value for the specified field from the provided struct.
// Supports nested access using dot notation (e.g., "user.address.street").
// Supports both struct fields and map keys.
func Get(origin any, field string) (any, error) {
	if origin == nil {
		return nil, fmt.Errorf("cannot get field %s from nil origin", field)
	}
	if field == "" {
		return nil, fmt.Errorf("field path cannot be empty")
	}

	tokens := strings.Split(field, ".")
	current := origin

	for i, token := range tokens {
		if token == "" {
			return nil, fmt.Errorf("empty token at position %d in field path %s", i, field)
		}

		valueOf := reflectutils.Deref(reflect.ValueOf(current))

		if !valueOf.IsValid() {
			return nil, fmt.Errorf("encountered nil value at token %s (position %d) in field path %s", token, i, field)
		}

		switch valueOf.Kind() {
		case reflect.Map:
			mapValue := valueOf.MapIndex(reflect.ValueOf(token))
			if !mapValue.IsValid() {
				return nil, fmt.Errorf("key %s not found in map at position %d in field path %s", token, i, field)
			}
			current = mapValue.Interface()

		case reflect.Struct:
			fieldValue := valueOf.FieldByName(token)
			if !fieldValue.IsValid() {
				return nil, fmt.Errorf("field %s not found in struct %s at position %d in field path %s", token, valueOf.Type().Name(), i, field)
			}
			if !fieldValue.CanInterface() {
				return nil, fmt.Errorf("field %s in struct %s is not exportable at position %d in field path %s", token, valueOf.Type().Name(), i, field)
			}
			current = fieldValue.Interface()

		default:
			return nil, fmt.Errorf("cannot traverse field %s: expected struct or map but got %s at position %d in field path %s", token, valueOf.Kind(), i, field)
		}
	}

	return current, nil
}
