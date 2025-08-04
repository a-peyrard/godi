package godi

import (
	"fmt"
	"github.com/a-peyrard/godi/fn"
	"github.com/a-peyrard/godi/option"
	"github.com/a-peyrard/godi/reflectutils"
	"github.com/a-peyrard/godi/structs"
	"reflect"
	"strings"
	"sync"
)

// ConfigDynamicProvider is a dynamic provider that provides all config fields as static providers.
type ConfigDynamicProvider[T any] struct {
	once          sync.Once
	names         []Name
	fieldWithType map[string]reflect.Type
	prefix        string
}

func (c *ConfigDynamicProvider[T]) CanBuild(name Name) bool {
	c.once.Do(func() {
		c.loadNames()
	})

	knownName, found := c.fieldWithType[name.name]
	return found && matchType(name.typ, knownName)
}

func (c *ConfigDynamicProvider[T]) BuildProviderFor(name Name) (provider Provider, opts []option.Option[RegisterOptions], err error) {
	/*
		Here we want to return a provider with the correct type, something like:
			func(cfg *T) (val FT, err error) {
		Where `FT` would be the type of the property in the config (we captured the types in fieldWithType).
		So to do that, we need to rely on reflection to build the correct signature.

		Maybe at some point we should allow having providers returning any, and skip the discovery of the type
		during register, but have some option to tell which type we are serving.
	*/
	// function signature:
	configType := reflect.TypeOf((*T)(nil))
	fnType := reflect.FuncOf(
		[]reflect.Type{configType},          // inject: config (T) as parameter
		[]reflect.Type{name.typ, ErrorType}, // output: fieldType (FT), error
		false,
	)

	// now we need to create a function that matches this signature
	fnImpl := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		cfg := args[0].Interface()

		value, err := structs.Get(cfg, strings.TrimPrefix(name.name, c.prefix))
		if err != nil {
			return []reflect.Value{
				reflect.Zero(name.typ),
				reflect.ValueOf(err),
			}
		}

		reflValue := reflect.ValueOf(value)
		if !reflValue.Type().AssignableTo(name.typ) {
			// the value is not the expected type, return an error
			return []reflect.Value{
				reflect.Zero(name.typ),
				reflect.ValueOf(fmt.Errorf("field %s has type %v, expected %v", name.name, reflValue.Type(), name.typ)),
			}
		}

		return []reflect.Value{reflValue, reflect.Zero(ErrorType)}
	})

	return fnImpl.Interface(),
		[]option.Option[RegisterOptions]{
			Named(name.name),
		},
		nil
}

func (c *ConfigDynamicProvider[T]) ListBuildableNames() []Name {
	c.once.Do(func() {
		c.loadNames()
	})
	return c.names
}

func (c *ConfigDynamicProvider[T]) loadNames() {
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
