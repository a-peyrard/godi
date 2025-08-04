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

func TestConfigDynamicProvider(t *testing.T) {
	t.Run("it should list all buildable names from config struct with correct types", func(t *testing.T) {
		// GIVEN
		provider := &ConfigDynamicProvider[TestConfig]{}

		// WHEN
		names := provider.ListBuildableNames()

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
		provider := &ConfigDynamicProvider[TestConfig]{}

		stringName := Name{name: "TestConfig.DatabaseURL", typ: reflect.TypeOf("")}
		intName := Name{name: "TestConfig.Port", typ: reflect.TypeOf(0)}
		nestedName := Name{name: "TestConfig.Nested.APIKey", typ: reflect.TypeOf("")}

		// WHEN & THEN
		assert.True(t, provider.CanBuild(stringName))
		assert.True(t, provider.CanBuild(intName))
		assert.True(t, provider.CanBuild(nestedName))
	})

	t.Run("it should return false for non-existent fields", func(t *testing.T) {
		// GIVEN
		provider := &ConfigDynamicProvider[TestConfig]{}
		nonExistentName := Name{name: "TestConfig.NonExistent", typ: reflect.TypeOf("")}

		// WHEN
		canBuild := provider.CanBuild(nonExistentName)

		// THEN
		assert.False(t, canBuild)
	})

	t.Run("it should return false for fields with wrong types", func(t *testing.T) {
		// GIVEN
		provider := &ConfigDynamicProvider[TestConfig]{}
		wrongTypeName := Name{name: "TestConfig.DatabaseURL", typ: reflect.TypeOf(0)} // DatabaseURL is string, not int

		// WHEN
		canBuild := provider.CanBuild(wrongTypeName)

		// THEN
		assert.False(t, canBuild)
	})

	t.Run("it should build working provider for string field", func(t *testing.T) {
		// GIVEN
		provider := &ConfigDynamicProvider[TestConfig]{}
		stringName := Name{name: "TestConfig.DatabaseURL", typ: reflect.TypeOf("")}
		testConfig := TestConfig{DatabaseURL: "postgres://localhost:5432/testdb"}

		// WHEN
		canBuild := provider.CanBuild(stringName)
		require.True(t, canBuild)
		providerFunc, opts, err := provider.BuildProviderFor(stringName)

		// THEN
		require.NoError(t, err)
		require.NotNil(t, providerFunc)
		require.Len(t, opts, 1)

		// Test the generated provider function
		funcValue := reflect.ValueOf(providerFunc)
		args := []reflect.Value{reflect.ValueOf(&testConfig)}
		results := funcValue.Call(args)

		require.Len(t, results, 2)
		assert.Equal(t, "postgres://localhost:5432/testdb", results[0].Interface())
		assert.True(t, results[1].IsNil()) // No error
	})

	t.Run("it should build working provider for int field", func(t *testing.T) {
		// GIVEN
		provider := &ConfigDynamicProvider[TestConfig]{}
		intName := Name{name: "TestConfig.Port", typ: reflect.TypeOf(0)}
		testConfig := TestConfig{Port: 8080}

		// WHEN
		canBuild := provider.CanBuild(intName)
		require.True(t, canBuild)
		providerFunc, _, err := provider.BuildProviderFor(intName)

		// THEN
		require.NoError(t, err)
		require.NotNil(t, providerFunc)

		// Test the generated provider function
		funcValue := reflect.ValueOf(providerFunc)
		args := []reflect.Value{reflect.ValueOf(&testConfig)}
		results := funcValue.Call(args)

		require.Len(t, results, 2)
		assert.Equal(t, 8080, results[0].Interface())
		assert.True(t, results[1].IsNil()) // No error
	})

	t.Run("it should build working provider for nested field", func(t *testing.T) {
		// GIVEN
		provider := &ConfigDynamicProvider[TestConfig]{}
		nestedName := Name{name: "TestConfig.Nested.APIKey", typ: reflect.TypeOf("")}
		testConfig := TestConfig{
			Nested: &NestedConfig{
				APIKey: "secret-key-123",
			},
		}

		// WHEN
		canBuild := provider.CanBuild(nestedName)
		require.True(t, canBuild)
		providerFunc, _, err := provider.BuildProviderFor(nestedName)

		// THEN
		require.NoError(t, err)
		require.NotNil(t, providerFunc)

		// Test the generated provider function
		funcValue := reflect.ValueOf(providerFunc)
		args := []reflect.Value{reflect.ValueOf(&testConfig)}
		results := funcValue.Call(args)

		require.Len(t, results, 2)
		assert.Equal(t, "secret-key-123", results[0].Interface())
		assert.True(t, results[1].IsNil()) // No error
	})

	t.Run("it should include correct options for built provider", func(t *testing.T) {
		// GIVEN
		provider := &ConfigDynamicProvider[TestConfig]{}
		fieldName := Name{name: "TestConfig.DatabaseURL", typ: reflect.TypeOf("")}

		// WHEN
		_, opts, err := provider.BuildProviderFor(fieldName)

		// THEN
		require.NoError(t, err)
		require.Len(t, opts, 1)

		// Apply options to see if Named is set correctly
		options := &RegisterOptions{}
		for _, opt := range opts {
			opt(options)
		}
		assert.Equal(t, "TestConfig.DatabaseURL", options.named)
	})

	t.Run("it should return false with CanBuild for non existing fields", func(t *testing.T) {
		// GIVEN
		provider := &ConfigDynamicProvider[TestConfig]{}
		nonExistentName := Name{name: "TestConfig.NonExistent", typ: reflect.TypeOf("")}

		// WHEN
		canBuild := provider.CanBuild(nonExistentName)

		// THEN
		assert.False(t, canBuild)
	})

	t.Run("it should cache names after first call", func(t *testing.T) {
		// GIVEN
		provider := &ConfigDynamicProvider[TestConfig]{}

		// WHEN
		names1 := provider.ListBuildableNames()
		names2 := provider.ListBuildableNames()

		// THEN
		assert.Equal(t, names1, names2)
		// Verify it's the same slice (cached)
		assert.Same(t, &names1[0], &names2[0])
	})
}
