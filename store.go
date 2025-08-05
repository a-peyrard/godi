package godi

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

type Store struct {
	inner sync.Map
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Put(name Name, comp reflect.Value) {
	s.inner.Store(name, comp)
}

func (s *Store) Get(name Name) (comp reflect.Value, found bool) {
	raw, found := s.inner.Load(name)
	if found {
		return raw.(reflect.Value), true
	}

	return reflect.Value{}, false
}

func (s *Store) Close() error {
	closeErrors := make([]error, 0)
	s.inner.Range(func(name, rawComp any) bool {
		comp := rawComp.(reflect.Value)
		if comp.IsValid() && comp.Type().Implements(CloseableType) {
			out := comp.MethodByName("Close").Call(nil)
			if len(out) != 1 || !out[0].IsNil() {
				closeErrors = append(
					closeErrors,
					fmt.Errorf("failed to close component %s:\n\t%v", name, out[0].Interface()),
				)
			}
		}
		return true // continue iteration
	})

	return errors.Join(closeErrors...)
}
