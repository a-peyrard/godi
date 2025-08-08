package godi

import (
	"fmt"
	"reflect"
)

var (
	StringType    = TypeOf[string]()
	ProviderType  = TypeOf[Provider]()
	DecoratorType = TypeOf[Decorator]()
	ErrorType     = TypeOf[error]()
	CloseableType = TypeOf[Closeable]()
	StringerType  = TypeOf[fmt.Stringer]()
)

func matchType(queryType, providedType reflect.Type) bool {
	if queryType == providedType {
		return true
	}
	if queryType.Kind() == reflect.Interface && providedType.Implements(queryType) {
		return true
	}
	return false
}

func TypeOf[I any]() reflect.Type {
	var i I
	t := reflect.TypeOf(i)
	if t == nil {
		t = reflect.TypeOf((*I)(nil)).Elem()
	}
	return t
}
