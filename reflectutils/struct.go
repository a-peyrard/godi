package reflectutils

import (
	"github.com/a-peyrard/godi/fn"
	"reflect"
)

// WalkStruct applies a bi-consumer on all fields and nested fields of a given object.
func WalkStruct[T any](element T, consumer fn.TriConsumer[reflect.Value, reflect.Type, []string]) {
	walkStructInternal(reflect.ValueOf(element), []string{}, consumer)
}

func walkStructInternal(val reflect.Value, path []string, consumer fn.TriConsumer[reflect.Value, reflect.Type, []string]) {
	var (
		nestedVal   reflect.Value
		structField reflect.StructField
	)
	// apply the consumer
	consumer(val, val.Type(), path)

	// dereference the value
	val = Deref(val)

	if !val.IsValid() {
		return
	}

	// loop on fields
	if val.Kind() == reflect.Struct {
		typ := val.Type()
		for i := 0; i < typ.NumField(); i++ {
			structField = typ.Field(i)
			if !structField.IsExported() {
				continue
			}
			nestedVal = val.Field(i)

			walkStructInternal(nestedVal, append(path, structField.Name), consumer)
		}
	}
}

// Deref dereferences recursively a reflect.Value until it reaches a non-pointer or non-interface value
func Deref(value reflect.Value) reflect.Value {
	if value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface {
		return Deref(value.Elem())
	}
	return value
}

// CreateNilStructs creates new struct instances for nil struct pointers
func CreateNilStructs(val reflect.Value, typ reflect.Type, _ []string) {
	if typ.Kind() == reflect.Pointer &&
		val.IsNil() &&
		typ.Elem().Kind() == reflect.Struct {

		val.Set(reflect.New(typ.Elem()))
	}
}

func CreateEmptyArrays(val reflect.Value, typ reflect.Type, _ []string) {
	if typ.Kind() == reflect.Slice && val.IsNil() {
		val.Set(reflect.MakeSlice(typ, 0, 0))
	}
}
