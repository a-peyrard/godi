package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for DI testing
type TestService struct {
	Name string
}

type TestRepository struct {
	Data string
}

type TestController struct {
	Service *TestService
	Repo    *TestRepository
}

// Provider functions for testing
func NewTestService() (*TestService, error) {
	return &TestService{Name: "test-service"}, nil
}

func NewTestRepository() (*TestRepository, error) {
	return &TestRepository{Data: "test-data"}, nil
}

func NewTestController(service *TestService, repo *TestRepository) (*TestController, error) {
	return &TestController{
		Service: service,
		Repo:    repo,
	}, nil
}

func NewFailingProvider() (*TestService, error) {
	return nil, errors.New("provider intentionally failed")
}

func TestResolver(t *testing.T) {
	t.Run("it should register a simple provider successfully", func(t *testing.T) {
		// GIVEN
		resolver := New()

		// WHEN
		err := resolver.Register(NewTestService)

		// THEN
		require.NoError(t, err)

		service, err := Resolve[*TestService](resolver)
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Equal(t, "test-service", service.Name)
	})

	t.Run("it should register multiple providers with dependencies", func(t *testing.T) {
		// GIVEN
		resolver := New()

		// WHEN
		err := resolver.Register(NewTestService)
		require.NoError(t, err)
		err = resolver.Register(NewTestRepository)
		require.NoError(t, err)
		err = resolver.Register(NewTestController)
		require.NoError(t, err)

		// THEN
		service, err := Resolve[*TestService](resolver)
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Equal(t, "test-service", service.Name)

		repo, err := Resolve[*TestRepository](resolver)
		require.NoError(t, err)
		assert.NotNil(t, repo)
		assert.Equal(t, "test-data", repo.Data)

		controller, err := Resolve[*TestController](resolver)
		require.NoError(t, err)
		assert.NotNil(t, controller)
		assert.NotNil(t, controller.Service)
		assert.NotNil(t, controller.Repo)
		assert.Equal(t, "test-service", controller.Service.Name)
		assert.Equal(t, "test-data", controller.Repo.Data)
	})

	t.Run("it should fail if provider is not a function", func(t *testing.T) {
		// GIVEN
		resolver := New()
		notAFunction := "this is not a function"

		// WHEN
		err := resolver.Register(notAFunction)

		// THEN
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider must be a function")
	})

	t.Run("it should fail if function returns wrong signature", func(t *testing.T) {
		// GIVEN
		resolver := New()

		// WHEN
		err := resolver.Register(func() string {
			return "not a valid provider"
		})

		// THEN
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider must return two values")
	})

	t.Run("it should return singleton instances (same instance on multiple resolves)", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewTestService)
		require.NoError(t, err)

		// WHEN
		service1, err := Resolve[*TestService](resolver)
		require.NoError(t, err)
		service2, err := Resolve[*TestService](resolver)
		require.NoError(t, err)

		// THEN
		assert.Same(t, service1, service2, "Expected same instance (singleton)")
	})

	t.Run("it should fail when no provider is registered for requested type", func(t *testing.T) {
		// GIVEN
		resolver := New()
		// No provider registered for TestService

		// WHEN
		_, err := Resolve[*TestService](resolver)

		// THEN
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no provider found")
	})

	t.Run("it should fail when provider function returns an error", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewFailingProvider)
		require.NoError(t, err)

		// WHEN
		_, err = Resolve[*TestService](resolver)

		// THEN
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider intentionally failed")
	})

	t.Run("it should fail when dependency cannot be resolved", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewTestController) // Depends on TestService and TestRepository
		require.NoError(t, err)
		// But TestService and TestRepository are not registered

		// WHEN
		_, err = Resolve[*TestController](resolver)

		// THEN
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve dependency")
	})

	// fixme: handle circular dependencies gracefully
	t.Run("it should handle circular dependencies gracefully", func(t *testing.T) {
		t.Skip() // fixme!

		// GIVEN
		resolver := New()

		// Create circular dependency providers
		circularProviderA := func(b *TestRepository) (*TestService, error) {
			return &TestService{Name: "circular-a"}, nil
		}
		circularProviderB := func(a *TestService) (*TestRepository, error) {
			return &TestRepository{Data: "circular-b"}, nil
		}

		err1 := resolver.Register(circularProviderA)
		err2 := resolver.Register(circularProviderB)
		require.NoError(t, err1)
		require.NoError(t, err2)

		// WHEN
		_, err := Resolve[*TestService](resolver)

		// THEN
		require.Error(t, err, "Expected error for circular dependency")
		// Note: This test might need adjustment based on how you want to handle circular deps
		// The current implementation might infinite loop or stack overflow
	})
}
