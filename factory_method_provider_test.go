package godi

import (
	"errors"
	"reflect"
	"testing"

	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for factory method testing
type TestDatabase struct {
	URL string
}

type TestLogger struct {
	Level string
}

func (t *TestLogger) Log(message string) {
	fmt.Println(t.Level, message)
}

type JustAnotherTestService struct {
	DB     *TestDatabase
	Logger *TestLogger
	Name   string
}

// Factory method providers for testing
func NewTestDatabase() (*TestDatabase, error) {
	return &TestDatabase{URL: "localhost:5432"}, nil
}

func NewTestLogger() (*TestLogger, error) {
	return &TestLogger{Level: "info"}, nil
}

func NewJustAnotherTestService(db *TestDatabase, logger *TestLogger) (*JustAnotherTestService, error) {
	return &JustAnotherTestService{
		DB:     db,
		Logger: logger,
		Name:   "test-service",
	}, nil
}

func NewFailingService() (*JustAnotherTestService, error) {
	return nil, errors.New("service creation failed")
}

func NewServiceWithoutError() *JustAnotherTestService {
	return &JustAnotherTestService{Name: "no-error-service"}
}

func TestFactoryMethodProvider(t *testing.T) {
	t.Run("it should create provider from simple factory method", func(t *testing.T) {
		// GIVEN
		factoryMethod := NewTestDatabase

		// WHEN
		provider, err := NewFactoryMethodProvider(factoryMethod)

		// THEN
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Verify provider properties
		names := provider.ListProvidableNames()
		require.Len(t, names, 1)
		assert.Equal(t, "godi.NewTestDatabase", names[0].name)
		assert.Equal(t, reflect.TypeOf(&TestDatabase{}), names[0].typ)
		assert.Equal(t, 0, provider.Priority())  // Default priority
		assert.Empty(t, provider.Dependencies()) // No dependencies
	})

	t.Run("it should create provider from factory method with dependencies", func(t *testing.T) {
		// GIVEN
		factoryMethod := NewJustAnotherTestService

		// WHEN
		provider, err := NewFactoryMethodProvider(factoryMethod)

		// THEN
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Verify provider properties
		names := provider.ListProvidableNames()
		require.Len(t, names, 1)
		assert.Equal(t, "godi.NewJustAnotherTestService", names[0].name)
		assert.Equal(t, reflect.TypeOf(&JustAnotherTestService{}), names[0].typ)

		deps := provider.Dependencies()
		require.Len(t, deps, 2) // db and logger dependencies
	})

	t.Run("it should create provider with custom options", func(t *testing.T) {
		// GIVEN
		factoryMethod := NewTestDatabase

		// WHEN
		provider, err := NewFactoryMethodProvider(
			factoryMethod,
			Named("custom.database"),
			Priority(100),
		)

		// THEN
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Verify custom options were applied
		names := provider.ListProvidableNames()
		require.Len(t, names, 1)
		assert.Equal(t, "custom.database", names[0].name)
		assert.Equal(t, 100, provider.Priority())
	})

	t.Run("it should reject non-function factory methods", func(t *testing.T) {
		// GIVEN
		notAFunction := "this is not a function"

		// WHEN
		provider, err := NewFactoryMethodProvider(notAFunction)

		// THEN
		require.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "factory method must be a function")
	})

	t.Run("it should reject factory methods with invalid return signature", func(t *testing.T) {
		// GIVEN
		invalidFactory := func() (int, string, error) {
			return 42, "invalid", nil
		}

		// WHEN
		provider, err := NewFactoryMethodProvider(invalidFactory)

		// THEN
		require.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "factory method must either return the instance and an error")
	})

	t.Run("it should correctly identify what it can provide", func(t *testing.T) {
		// GIVEN
		provider, err := NewFactoryMethodProvider(NewTestDatabase)
		require.NoError(t, err)

		correctName := Name{name: "godi.NewTestDatabase", typ: reflect.TypeOf(&TestDatabase{})}
		wrongName := Name{name: "godi.WrongName", typ: reflect.TypeOf(&TestDatabase{})}
		wrongType := Name{name: "godi.NewTestDatabase", typ: reflect.TypeOf(&TestLogger{})}

		// WHEN & THEN
		assert.True(t, provider.CanProvide(correctName))
		assert.False(t, provider.CanProvide(wrongName))
		assert.False(t, provider.CanProvide(wrongType))
	})

	t.Run("it should provide instance successfully with no dependencies", func(t *testing.T) {
		// GIVEN
		provider, err := NewFactoryMethodProvider(NewTestDatabase)
		require.NoError(t, err)

		targetName := Name{name: "NewTestDatabase", typ: reflect.TypeOf(&TestDatabase{})}

		// WHEN
		instance, err := provider.Provide(targetName, []reflect.Value{})

		// THEN
		require.NoError(t, err)
		require.True(t, instance.IsValid())

		db, ok := instance.Interface().(*TestDatabase)
		require.True(t, ok)
		assert.Equal(t, "localhost:5432", db.URL)
	})

	t.Run("it should provide instance successfully with dependencies", func(t *testing.T) {
		// GIVEN
		provider, err := NewFactoryMethodProvider(NewJustAnotherTestService)
		require.NoError(t, err)

		// Create mock dependencies
		mockDB := &TestDatabase{URL: "mock-db"}
		mockLogger := &TestLogger{Level: "debug"}
		dependencies := []reflect.Value{
			reflect.ValueOf(mockDB),
			reflect.ValueOf(mockLogger),
		}

		targetName := Name{name: "NewJustAnotherTestService", typ: reflect.TypeOf(&JustAnotherTestService{})}

		// WHEN
		instance, err := provider.Provide(targetName, dependencies)

		// THEN
		require.NoError(t, err)
		require.True(t, instance.IsValid())

		service, ok := instance.Interface().(*JustAnotherTestService)
		require.True(t, ok)
		assert.Equal(t, "test-service", service.Name)
		assert.Same(t, mockDB, service.DB)
		assert.Same(t, mockLogger, service.Logger)
	})

	t.Run("it should handle factory method that returns error", func(t *testing.T) {
		// GIVEN
		provider, err := NewFactoryMethodProvider(NewFailingService)
		require.NoError(t, err)

		targetName := Name{name: "NewFailingService", typ: reflect.TypeOf(&JustAnotherTestService{})}

		// WHEN
		instance, err := provider.Provide(targetName, []reflect.Value{})

		// THEN
		require.Error(t, err)
		assert.False(t, instance.IsValid())
		assert.Contains(t, err.Error(), "service creation failed")
	})

	t.Run("it should handle factory method without error return", func(t *testing.T) {
		// GIVEN
		provider, err := NewFactoryMethodProvider(NewServiceWithoutError)
		require.NoError(t, err)

		targetName := Name{name: "NewServiceWithoutError", typ: reflect.TypeOf(&JustAnotherTestService{})}

		// WHEN
		instance, err := provider.Provide(targetName, []reflect.Value{})

		// THEN
		require.NoError(t, err)
		require.True(t, instance.IsValid())

		service, ok := instance.Interface().(*JustAnotherTestService)
		require.True(t, ok)
		assert.Equal(t, "no-error-service", service.Name)
	})

	t.Run("it should handle panic in factory method gracefully", func(t *testing.T) {
		// GIVEN
		panickyFactory := func() (*JustAnotherTestService, error) {
			panic("something went wrong")
		}
		provider, err := NewFactoryMethodProvider(panickyFactory)
		require.NoError(t, err)

		targetName := Name{name: "main.TestFactoryMethodProvider.func1", typ: reflect.TypeOf(&JustAnotherTestService{})}

		// WHEN
		instance, err := provider.Provide(targetName, []reflect.Value{})

		// THEN
		require.Error(t, err)
		assert.False(t, instance.IsValid())
		assert.Contains(t, err.Error(), "panic calling provider")
		assert.Contains(t, err.Error(), "something went wrong")
	})
}
