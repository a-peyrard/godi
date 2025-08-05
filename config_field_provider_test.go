package godi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test config structs
type TestConfig struct {
	DatabaseURL string
	Port        int
	Debug       bool
	Timeout     float64
	Nested      *NestedConfig
}

type NestedConfig struct {
	APIKey     string
	MaxRetries int
}

func TestConfigFieldProvider(t *testing.T) {
	t.Run("it should list all buildable names from config struct with correct types", func(t *testing.T) {
		// GIVEN
		provider := &ConfigFieldProvider[TestConfig]{}

		// WHEN
		names := provider.ListProvidableNames()

		// THEN
		require.NotEmpty(t, names)
		require.Len(t, names, 7) // 5 fields + 2 inside the nested struct

		// Check that all expected field names are present
		typeMap := make(map[string]reflect.Type)
		for _, name := range names {
			typeMap[name.name] = name.typ
		}

		assert.Equal(t, reflect.TypeOf(""), typeMap["TestConfig.DatabaseURL"])
		assert.Equal(t, reflect.TypeOf(0), typeMap["TestConfig.Port"])
		assert.Equal(t, reflect.TypeOf(false), typeMap["TestConfig.Debug"])
		assert.Equal(t, reflect.TypeOf(0.0), typeMap["TestConfig.Timeout"])
		assert.Equal(t, reflect.TypeOf(&NestedConfig{}), typeMap["TestConfig.Nested"])
		assert.Equal(t, reflect.TypeOf(""), typeMap["TestConfig.Nested.APIKey"])
		assert.Equal(t, reflect.TypeOf(0), typeMap["TestConfig.Nested.MaxRetries"])
	})

	t.Run("it should return true for buildable fields with correct types", func(t *testing.T) {
		// GIVEN
		provider := &ConfigFieldProvider[TestConfig]{}

		stringName := Name{name: "TestConfig.DatabaseURL", typ: reflect.TypeOf("")}
		intName := Name{name: "TestConfig.Port", typ: reflect.TypeOf(0)}
		nestedName := Name{name: "TestConfig.Nested.APIKey", typ: reflect.TypeOf("")}

		// WHEN & THEN
		assert.True(t, provider.CanProvide(stringName))
		assert.True(t, provider.CanProvide(intName))
		assert.True(t, provider.CanProvide(nestedName))
	})

	t.Run("it should return false for non-existent fields", func(t *testing.T) {
		// GIVEN
		provider := &ConfigFieldProvider[TestConfig]{}
		nonExistentName := Name{name: "TestConfig.NonExistent", typ: reflect.TypeOf("")}

		// WHEN
		canProvide := provider.CanProvide(nonExistentName)

		// THEN
		assert.False(t, canProvide)
	})

	t.Run("it should return false for fields with wrong types", func(t *testing.T) {
		// GIVEN
		provider := &ConfigFieldProvider[TestConfig]{}
		wrongTypeName := Name{name: "TestConfig.DatabaseURL", typ: reflect.TypeOf(0)} // DatabaseURL is string, not int

		// WHEN
		canProvide := provider.CanProvide(wrongTypeName)

		// THEN
		assert.False(t, canProvide)
	})

	t.Run("it should build component for string field", func(t *testing.T) {
		// GIVEN
		provider := &ConfigFieldProvider[TestConfig]{}
		stringName := Name{name: "TestConfig.DatabaseURL", typ: reflect.TypeOf("")}
		testConfig := &TestConfig{DatabaseURL: "postgres://localhost:5432/testdb"}

		// WHEN
		canProvide := provider.CanProvide(stringName)
		require.True(t, canProvide)
		val, err := provider.Provide(stringName, []reflect.Value{reflect.ValueOf(testConfig)})

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "postgres://localhost:5432/testdb", val.Interface())
	})

	t.Run("it should build component for int field", func(t *testing.T) {
		// GIVEN
		provider := &ConfigFieldProvider[TestConfig]{}
		intName := Name{name: "TestConfig.Port", typ: reflect.TypeOf(0)}
		testConfig := &TestConfig{Port: 8080}

		// WHEN
		canProvide := provider.CanProvide(intName)
		require.True(t, canProvide)
		val, err := provider.Provide(intName, []reflect.Value{reflect.ValueOf(testConfig)})

		// THEN
		require.NoError(t, err)
		assert.Equal(t, 8080, val.Interface())
	})

	t.Run("it should build component for nested field", func(t *testing.T) {
		// GIVEN
		provider := &ConfigFieldProvider[TestConfig]{}
		nestedName := Name{name: "TestConfig.Nested.APIKey", typ: reflect.TypeOf("")}
		testConfig := &TestConfig{
			Nested: &NestedConfig{
				APIKey: "secret-key-123",
			},
		}

		// WHEN
		canProvide := provider.CanProvide(nestedName)
		require.True(t, canProvide)
		val, err := provider.Provide(nestedName, []reflect.Value{reflect.ValueOf(testConfig)})

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "secret-key-123", val.Interface())
	})

	t.Run("it should return false with CanProvide for non existing fields", func(t *testing.T) {
		// GIVEN
		provider := &ConfigFieldProvider[TestConfig]{}
		nonExistentName := Name{name: "TestConfig.NonExistent", typ: reflect.TypeOf("")}

		// WHEN
		canProvide := provider.CanProvide(nonExistentName)

		// THEN
		assert.False(t, canProvide)
	})

	t.Run("it should cache names after first call", func(t *testing.T) {
		// GIVEN
		provider := &ConfigFieldProvider[TestConfig]{}

		// WHEN
		names1 := provider.ListProvidableNames()
		names2 := provider.ListProvidableNames()

		// THEN
		assert.Equal(t, names1, names2)
		// Verify it's the same slice (cached)
		assert.Same(t, &names1[0], &names2[0])
	})
}
