package godi

import (
	"fmt"
	"reflect"
)

type (
	query interface {
		want(name Name) bool

		fmt.Stringer
	}

	queryByType struct {
		typ reflect.Type
	}

	queryByName struct {
		name Name
	}
)

func (q queryByType) want(n Name) bool {
	return matchType(q.typ, n.typ)
}

func (q queryByType) String() string {
	return fmt.Sprintf("<type ~= %s>", q.typ.String())
}

func (q queryByName) want(n Name) bool {
	return n.name == q.name.name && matchType(q.name.typ, n.typ)
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
