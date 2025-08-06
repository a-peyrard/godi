package godi

import (
	"os"
	"reflect"
	"strings"
	"sync"
)

// EnvProvider is a provider that provides environment variables as components.
type EnvProvider struct {
	once  sync.Once
	names []Name
}

func (e *EnvProvider) CanProvide(name Name) bool {
	if name.typ == StringType && name.name != "" {
		_, found := os.LookupEnv(name.name)
		if found {
			return true
		}
	}

	return false
}

func (e *EnvProvider) Provide(name Name, _ []reflect.Value) (comp reflect.Value, err error) {
	return reflect.ValueOf(os.Getenv(name.name)), nil
}

func (e *EnvProvider) Dependencies() []Request {
	return nil
}

func (e *EnvProvider) ListProvidableNames() []Name {
	e.once.Do(func() {
		e.loadNames()
	})
	return e.names
}

func (e *EnvProvider) Priority() int {
	return 0
}

func (e *EnvProvider) ListBuildableNames() []Name {
	e.once.Do(func() {
		e.loadNames()
	})
	return e.names
}

func (e *EnvProvider) loadNames() {
	props := os.Environ()
	e.names = make([]Name, len(props))
	for i, prop := range props {
		tokens := strings.SplitN(prop, "=", 2)
		e.names[i] = Name{
			name: tokens[0],
			typ:  StringType,
		}
	}
}

func (e *EnvProvider) Description() string {
	return "Provides environment variables as string components"
}
