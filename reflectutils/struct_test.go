package reflectutils

import (
	"github.com/a-peyrard/godi/fn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"strings"
	"testing"
)

type (
	TestConfig struct {
		Foo       *FooTestConfig
		Bar       *BarTestConfig
		SomeValue string
	}
	TestConfigWithPrivate struct {
		Foo       *FooTestConfig
		Bar       *BarTestConfig
		foobar    *FooBarTestConfig
		SomeValue string
	}
	FooTestConfig struct {
		Hello string
		World int
	}
	BarTestConfig struct {
		First  int
		Second int
	}
	FooBarTestConfig struct {
		waldo int
	}
	WithDefault interface {
		ApplyDefault()
	}
	TestConfigWithArray struct {
		SomeValue []string
	}
)

func (c *TestConfig) ApplyDefault() {
	if c.SomeValue == "" {
		c.SomeValue = "hello world"
	}
}

func (c *BarTestConfig) ApplyDefault() {
	if c.First == 0 {
		c.First = 42
	}
}

func TestWalkStruct(t *testing.T) {
	t.Run("it should apply consumer on all fields", func(t *testing.T) {
		// GIVEN
		// create a consumer calling the apply default method if the struct is implementing WithDefault
		consumer := func(val reflect.Value, typ reflect.Type) {
			withDefaultValueType := reflect.TypeOf((*WithDefault)(nil)).Elem()
			if typ.Implements(withDefaultValueType) {
				if val.IsValid() {
					val.Interface().(WithDefault).ApplyDefault()
				}
			}
		}

		// WHEN
		element := &TestConfig{
			Foo: &FooTestConfig{},
			Bar: &BarTestConfig{},
		}
		WalkStruct(element, consumer)

		// THEN
		assert.Equal(t, "hello world", element.SomeValue)
		assert.Equal(t, 42, element.Bar.First)
	})

	t.Run("it should allow to initialize sub structs", func(t *testing.T) {
		// WHEN
		element := &TestConfig{}
		WalkStruct(element, CreateNilStructs)

		// THEN
		assert.Equal(t, "", element.SomeValue)
		require.NotNil(t, element.Foo)
		assert.Equal(t, "", element.Foo.Hello)
		assert.Equal(t, 0, element.Foo.World)
		require.NotNil(t, element.Bar)
		assert.Equal(t, 0, element.Bar.First)
		assert.Equal(t, 0, element.Bar.Second)
	})

	t.Run("it should ignore private fields when initializing sub structs", func(t *testing.T) {
		// WHEN
		element := &TestConfigWithPrivate{}
		WalkStruct(element, CreateNilStructs)

		// THEN
		assert.Equal(t, "", element.SomeValue)
		require.NotNil(t, element.Foo)
		assert.Equal(t, "", element.Foo.Hello)
		assert.Equal(t, 0, element.Foo.World)
		require.NotNil(t, element.Bar)
		assert.Equal(t, 0, element.Bar.First)
		assert.Equal(t, 0, element.Bar.Second)
		assert.Nil(t, element.foobar)
	})

	t.Run("it should deref pointer of interfaces", func(t *testing.T) {
		// WHEN
		element := &TestConfigWithPrivate{}
		var iface any = element
		var ptrIface any = &iface
		WalkStruct(ptrIface, CreateNilStructs)

		// THEN
		assert.Equal(t, "", element.SomeValue)
		require.NotNil(t, element.Foo)
		assert.Equal(t, "", element.Foo.Hello)
		assert.Equal(t, 0, element.Foo.World)
		require.NotNil(t, element.Bar)
		assert.Equal(t, 0, element.Bar.First)
		assert.Equal(t, 0, element.Bar.Second)
		assert.Nil(t, element.foobar)
	})

	t.Run("it should allow to initialize sub structs and also apply default", func(t *testing.T) {
		// GIVEN
		// create a consumer calling the apply default method if the struct is implementing WithDefault
		withDefaultValueType := reflect.TypeOf((*WithDefault)(nil)).Elem()
		callApplyDefault := func(val reflect.Value, typ reflect.Type) {
			if typ.Implements(withDefaultValueType) {
				if val.IsValid() {
					val.Interface().(WithDefault).ApplyDefault()
				}
			}
		}

		// WHEN
		element := &TestConfig{}
		WalkStruct(element, fn.AllBiConsumer(CreateNilStructs, callApplyDefault))

		// THEN
		assert.Equal(t, "hello world", element.SomeValue)
		require.NotNil(t, element.Foo)
		assert.Equal(t, "", element.Foo.Hello)
		assert.Equal(t, 0, element.Foo.World)
		require.NotNil(t, element.Bar)
		assert.Equal(t, 42, element.Bar.First)
		assert.Equal(t, 0, element.Bar.Second)
	})

	t.Run("it should not recurse on invalid ref", func(t *testing.T) {
		// GIVEN
		type Foo struct {
			Bar string
		}
		type Test struct {
			Foo *Foo
		}

		nilFoo := func(val reflect.Value, typ reflect.Type) {
			if strings.Contains(typ.String(), "Foo") {
				val.Set(reflect.Zero(typ))
			}
		}

		// WHEN
		element := &Test{
			Foo: &Foo{
				Bar: "hello",
			},
		}
		WalkStruct(element, nilFoo)

		// THEN
		assert.Nil(t, element.Foo)
	})

	t.Run("it should allow to initialize nil arrays", func(t *testing.T) {
		// WHEN
		element := &TestConfigWithArray{}
		WalkStruct(element, CreateEmptyArrays)

		// THEN
		require.NotNil(t, element.SomeValue)
		assert.Equal(t, []string{}, element.SomeValue)
	})
}
