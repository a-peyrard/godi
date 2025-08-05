package godi

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/a-peyrard/godi/fn"
	"github.com/a-peyrard/godi/reflectutils"
	"github.com/a-peyrard/godi/structs"
)

// ConfigFieldProvider is a provider that provides all config fields as components.
type ConfigFieldProvider[T any] struct {
	once          sync.Once
	names         []Name
	fieldWithType map[string]reflect.Type
	prefix        string
}

func (c *ConfigFieldProvider[T]) CanProvide(name Name) bool {
	c.loadNamesIfNeeded()

	knownName, found := c.fieldWithType[name.name]
	return found && matchType(name.typ, knownName)
}

func (c *ConfigFieldProvider[T]) Provide(name Name, dependencies []reflect.Value) (comp reflect.Value, err error) {
	cfg := dependencies[0].Interface()

	value, err := structs.Get(cfg, strings.TrimPrefix(name.name, c.prefix))
	if err != nil {
		return reflect.Zero(name.typ), err
	}

	reflValue := reflect.ValueOf(value)
	if !reflValue.Type().AssignableTo(name.typ) {
		// the value is not the expected type, return an error
		return reflect.Zero(name.typ), fmt.Errorf("field %s has type %v, expected %v", name.name, reflValue.Type(), name.typ)
	}

	return reflValue, nil
}

func (c *ConfigFieldProvider[T]) Dependencies() []Request {
	configType := reflect.TypeOf((*T)(nil))
	return []Request{
		{
			unitaryTyp: configType,
			query:      queryByType{typ: configType},
			validator:  validatorUniqueMandatory{},
			collector:  collectorUnique{},
		},
	}
}

func (c *ConfigFieldProvider[T]) ListProvidableNames() []Name {
	c.loadNamesIfNeeded()
	return c.names
}

func (c *ConfigFieldProvider[T]) Priority() int {
	return 0
}

func (c *ConfigFieldProvider[T]) loadNamesIfNeeded() {
	c.once.Do(func() {
		c.loadNamesInternal()
	})
}

func (c *ConfigFieldProvider[T]) loadNamesInternal() {
	emptyConfig := new(T)
	// we prefix all providers by the config struct name,
	// so if one want to get the value of the field "Port" in the struct "TestConfig",
	// the provider will be named "TestConfig.Port".
	c.prefix = reflect.TypeOf(emptyConfig).Elem().Name() + "."

	reflectutils.WalkStruct(emptyConfig, reflectutils.CreateNilStructs)

	c.fieldWithType = make(map[string]reflect.Type)
	reflectutils.WalkStruct(
		emptyConfig,
		fn.AllTriConsumer(
			reflectutils.CreateNilStructs,
			func(_ reflect.Value, fieldTyp reflect.Type, path []string) {
				if len(path) > 0 {
					fieldPath := c.prefix + strings.Join(path, ".")
					c.fieldWithType[fieldPath] = fieldTyp
				}
			},
		),
	)

	c.names = make([]Name, 0, len(c.fieldWithType))
	for fieldPath, fieldTyp := range c.fieldWithType {
		c.names = append(
			c.names,
			Name{
				name: fieldPath,
				typ:  fieldTyp,
			},
		)
	}
}
