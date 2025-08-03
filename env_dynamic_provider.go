package godi

import (
	"github.com/a-peyrard/godi/option"
	"os"
	"strings"
	"sync"
)

// EnvDynamicProvider is a dynamic provider that provides environment variables as static providers.
type EnvDynamicProvider struct {
	once  sync.Once
	names []Name
}

func (e *EnvDynamicProvider) CanBuild(name Name) bool {
	if name.typ == StringType && name.name != "" {
		_, found := os.LookupEnv(name.name)
		if found {
			return true
		}
	}

	return false
}

func (e *EnvDynamicProvider) BuildProviderFor(name Name) (provider Provider, opts []option.Option[RegisterOptions], err error) {
	return ToStaticProvider(os.Getenv(name.name)), []option.Option[RegisterOptions]{Named(name.name)}, nil
}

func (e *EnvDynamicProvider) ListBuildableNames() []Name {
	e.once.Do(func() {
		e.loadNames()
	})
	return e.names
}

func (e *EnvDynamicProvider) loadNames() {
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
