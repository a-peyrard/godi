package godi

import "reflect"

var StringType = reflect.TypeOf("")
var DynamicProviderType = reflect.TypeOf((*DynamicProvider)(nil)).Elem()
var ErrorType = reflect.TypeOf((*error)(nil)).Elem()
