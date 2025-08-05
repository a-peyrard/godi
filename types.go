package godi

import "reflect"

var StringType = reflect.TypeOf("")
var ProviderType = reflect.TypeOf((*Provider)(nil)).Elem()
var ErrorType = reflect.TypeOf((*error)(nil)).Elem()
var CloseableType = reflect.TypeOf((*Closeable)(nil)).Elem()

func matchType(queryType, providedType reflect.Type) bool {
	if queryType == providedType {
		return true
	}
	if queryType.Kind() == reflect.Interface && providedType.Implements(queryType) {
		return true
	}
	return false
}
