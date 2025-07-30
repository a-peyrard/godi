package di

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var closeCounter atomic.Int32

// Test types for DI testing
type TestService struct {
	Name   string
	closed bool
}

func (t *TestService) Close() error {
	t.closed = true
	closeCounter.Add(1)
	return nil
}

type TestRepository struct {
	Data   string
	closed bool
}

func (t *TestRepository) Close() error {
	t.closed = true
	closeCounter.Add(1)
	return nil
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

	t.Run("it should not care about registering order when resolving dependencies", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewTestController)
		require.NoError(t, err)
		err = resolver.Register(NewTestService)
		require.NoError(t, err)
		err = resolver.Register(NewTestRepository)
		require.NoError(t, err)

		// WHEN
		controller, err := Resolve[*TestController](resolver)

		// THEN
		require.NoError(t, err)

		require.NoError(t, err)
		assert.NotNil(t, controller)
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

	t.Run("it should fail if multiple providers can resolve the same requirement", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(func() (*TestService, error) {
			return &TestService{Name: "test-service-1"}, nil
		})
		require.NoError(t, err)
		err = resolver.Register(func() (*TestService, error) {
			return &TestService{Name: "test-service-2"}, nil
		})
		require.NoError(t, err)

		// WHEN
		_, err = Resolve[*TestService](resolver)

		// THEN
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multiple providers found for query")
	})

	t.Run("it should allow to resolve all providers for a given type", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(func() (*TestService, error) {
			return &TestService{Name: "test-service-1"}, nil
		})
		require.NoError(t, err)
		err = resolver.Register(func() (*TestService, error) {
			return &TestService{Name: "test-service-2"}, nil
		})
		require.NoError(t, err)

		// WHEN
		resolved, err := ResolveAll[*TestService](resolver)

		// THEN
		require.NoError(t, err)
		assert.Len(t, resolved, 2)
		names := []string{resolved[0].Name, resolved[1].Name}
		assert.Contains(t, names, "test-service-1")
		assert.Contains(t, names, "test-service-2")
	})

	t.Run("it should allow to resolve by interface and get implementing types", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewTestService)
		require.NoError(t, err)
		err = resolver.Register(NewTestRepository)
		require.NoError(t, err)
		err = resolver.Register(NewTestController)
		require.NoError(t, err)

		// WHEN
		resolved, err := ResolveAll[io.Closer](resolver)

		// THEN
		require.NoError(t, err)
		assert.Len(t, resolved, 2)
		types := []string{fmt.Sprintf("%T", resolved[0]), fmt.Sprintf("%T", resolved[1])}
		assert.Contains(t, types, "*di.TestService")
		assert.Contains(t, types, "*di.TestRepository")
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

func TestResolver_Close(t *testing.T) {
	t.Run("it should close all instantiated closeable when closing resolver", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewTestService)
		require.NoError(t, err)
		err = resolver.Register(NewTestRepository)
		require.NoError(t, err)
		err = resolver.Register(NewTestController)
		require.NoError(t, err)

		testService, err := Resolve[*TestService](resolver)
		require.NoError(t, err)
		testRepository, err := Resolve[*TestRepository](resolver)
		require.NoError(t, err)

		// WHEN
		err = resolver.Close()

		// THEN
		require.NoError(t, err)
		assert.True(t, testService.closed)
		assert.True(t, testRepository.closed)
	})

	t.Run("it should close only instantiated providers", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewTestService)
		require.NoError(t, err)
		err = resolver.Register(NewTestRepository)
		require.NoError(t, err)
		err = resolver.Register(NewTestController)
		require.NoError(t, err)

		_, err = Resolve[*TestService](resolver)
		require.NoError(t, err)

		// WHEN
		// the counter is not ideal, as it would not work if we start running tests in parallel
		// but as long as we run tests sequentially, it should be fine
		before := closeCounter.Load()
		err = resolver.Close()
		require.NoError(t, err)
		after := closeCounter.Load()

		// THEN
		assert.Equal(t, int32(1), after-before)
	})
}

var runCounter atomic.Int32

type (
	Coyote struct{}

	LoadRunner struct{}

	ContextRunner struct {
		Hello string
	}

	greetKey struct{}
)

func (c *Coyote) Run(context.Context) error {
	runCounter.Add(1)
	return nil
}

func (l *LoadRunner) Run(context.Context) error {
	runCounter.Add(1)
	return nil
}

func (l *ContextRunner) Run(ctx context.Context) error {
	val := ctx.Value(greetKey{}).(string)
	if val == "" {
		l.Hello = "Waldo"
	} else {
		l.Hello = val
	}

	return nil
}

func TestResolver_Run(t *testing.T) {
	t.Run("it should run all runnables", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(func() (*Coyote, error) {
			return &Coyote{}, nil
		})
		require.NoError(t, err)
		err = resolver.Register(func() (*LoadRunner, error) {
			return &LoadRunner{}, nil
		})
		require.NoError(t, err)
		err = resolver.Register(NewTestController)

		// WHEN
		startingRunCount := runCounter.Load()
		err = resolver.Run()
		require.NoError(t, err)
		endingRunCount := runCounter.Load()

		// THEN
		require.NoError(t, err)
		assert.Equal(t, int32(2), endingRunCount-startingRunCount)
	})

	t.Run("it should use provided context if one is provided", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(func() (context.Context, error) {
			ctx := context.WithValue(t.Context(), greetKey{}, "Augustin")
			return ctx, nil
		})
		require.NoError(t, err)
		err = resolver.Register(func() (*ContextRunner, error) {
			return &ContextRunner{}, nil
		})
		require.NoError(t, err)

		// WHEN
		err = resolver.Run()
		require.NoError(t, err)

		// THEN
		contextRunner, err := Resolve[*ContextRunner](resolver)
		require.NoError(t, err)
		assert.Equal(t, "Augustin", contextRunner.Hello)
	})
}

func TestResolver_TryResolve(t *testing.T) {
	t.Run("it should return found=true when component exists", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewTestService)
		require.NoError(t, err)

		// WHEN
		service, found, err := TryResolve[*TestService](resolver)

		// THEN
		require.NoError(t, err)
		assert.True(t, found)
		require.NotNil(t, service)
		assert.Equal(t, "test-service", service.Name)
	})

	t.Run("it should return found=false when component does not exist", func(t *testing.T) {
		// GIVEN
		resolver := New()

		// WHEN
		service, found, err := TryResolve[*TestService](resolver)

		// THEN
		require.NoError(t, err)
		assert.False(t, found)
		assert.Nil(t, service)
	})

	t.Run("it should return error when provider function fails", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewFailingProvider)
		require.NoError(t, err)

		// WHEN
		service, found, err := TryResolve[*TestService](resolver)

		// THEN
		require.Error(t, err)
		assert.False(t, found)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "provider intentionally failed")
	})

	t.Run("it should return error when dependency cannot be resolved", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewTestController) // Depends on TestService and TestRepository
		require.NoError(t, err)
		// But TestService and TestRepository are not registered

		// WHEN
		controller, found, err := TryResolve[*TestController](resolver)

		// THEN
		require.Error(t, err)
		assert.False(t, found)
		assert.Nil(t, controller)
		assert.Contains(t, err.Error(), "failed to resolve dependency")
	})

	t.Run("it should resolve complex dependencies when all are available", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewTestService)
		require.NoError(t, err)
		err = resolver.Register(NewTestRepository)
		require.NoError(t, err)
		err = resolver.Register(NewTestController)
		require.NoError(t, err)

		// WHEN
		controller, found, err := TryResolve[*TestController](resolver)

		// THEN
		require.NoError(t, err)
		assert.True(t, found)
		require.NotNil(t, controller)
		require.NotNil(t, controller.Service)
		require.NotNil(t, controller.Repo)
		assert.Equal(t, "test-service", controller.Service.Name)
		assert.Equal(t, "test-data", controller.Repo.Data)
	})

	t.Run("it should return same instance as Resolve for existing components", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewTestService)
		require.NoError(t, err)

		// WHEN
		service1, err := Resolve[*TestService](resolver)
		require.NoError(t, err)
		service2, found, err := TryResolve[*TestService](resolver)

		// THEN
		require.NoError(t, err)
		assert.True(t, found)
		assert.Same(t, service1, service2, "TryResolve should return same singleton instance")
	})
}

type (
	NameSupplier struct {
		name string
	}
)

func (n *NameSupplier) Name() string {
	return n.name
}

func TestResolver_Register(t *testing.T) {
	t.Run("it should allows to register with custom name", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(
			func() (*NameSupplier, error) {
				return &NameSupplier{name: "Augustin"}, nil
			},
			Named("firstName"),
		)
		require.NoError(t, err)
		err = resolver.Register(
			func() (*NameSupplier, error) {
				return &NameSupplier{name: "Peyrard"}, nil
			},
			Named("lastName"),
		)
		require.NoError(t, err)

		// WHEN
		names, err := ResolveAll[*NameSupplier](resolver)
		require.NoError(t, err)
		firstName, err := ResolveNamed[*NameSupplier](resolver, "firstName")
		lastName, err := ResolveNamed[*NameSupplier](resolver, "lastName")

		// THEN
		require.NoError(t, err)
		assert.Len(t, names, 2)
		namesFound := []string{names[0].Name(), names[1].Name()}
		assert.Contains(t, namesFound, "Augustin")
		assert.Contains(t, namesFound, "Peyrard")

		assert.Equal(t, "Augustin", firstName.Name())
		assert.Equal(t, "Peyrard", lastName.Name())
	})

	t.Run("it should allows to register with custom priority and take precedence when resolving", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(
			func() (*NameSupplier, error) {
				return &NameSupplier{name: "Peyrard"}, nil
			},
			Named("lastName"),
		)
		require.NoError(t, err)
		err = resolver.Register(
			func() (*NameSupplier, error) {
				return &NameSupplier{name: "Arshinov"}, nil
			},
			Named("lastName"),
			Priority(100),
		)
		err = resolver.Register(
			func() (*NameSupplier, error) {
				return &NameSupplier{name: "Waldo"}, nil
			},
			Named("lastName"),
			Priority(10),
		)
		require.NoError(t, err)

		// WHEN
		name, err := Resolve[*NameSupplier](resolver)

		// THEN
		require.NoError(t, err)

		assert.Equal(t, "Arshinov", name.Name())
	})

	t.Run("it should allows to register with custom priority and take precedence when using named resolution", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(
			func() (*NameSupplier, error) {
				return &NameSupplier{name: "Peyrard"}, nil
			},
			Named("lastName"),
		)
		require.NoError(t, err)
		err = resolver.Register(
			func() (*NameSupplier, error) {
				return &NameSupplier{name: "Arshinov"}, nil
			},
			Named("lastName"),
			Priority(100),
		)
		err = resolver.Register(
			func() (*NameSupplier, error) {
				return &NameSupplier{name: "Waldo"}, nil
			},
			Named("lastName"),
			Priority(10),
		)
		require.NoError(t, err)

		// WHEN
		name, err := ResolveNamed[*NameSupplier](resolver, "lastName")

		// THEN
		require.NoError(t, err)

		assert.Equal(t, "Arshinov", name.Name())
	})

	t.Run("it should resolve only the highest priority when resolving all", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(
			func() (*NameSupplier, error) {
				return &NameSupplier{name: "Peyrard"}, nil
			},
			Named("lastName"),
		)
		require.NoError(t, err)
		err = resolver.Register(
			func() (*NameSupplier, error) {
				return &NameSupplier{name: "Arshinov"}, nil
			},
			Named("lastName"),
			Priority(100),
		)
		err = resolver.Register(
			func() (*NameSupplier, error) {
				return &NameSupplier{name: "Waldo"}, nil
			},
			Named("lastName"),
			Priority(10),
		)
		require.NoError(t, err)

		// WHEN
		names, err := ResolveAll[*NameSupplier](resolver)

		// THEN
		require.NoError(t, err)
		assert.Len(t, names, 1)
		assert.Equal(t, "Arshinov", names[0].Name())
	})
}

func TestResolver_MustRegister(t *testing.T) {
	t.Run("it should register provider successfully and return resolver for chaining", func(t *testing.T) {
		// GIVEN
		resolver := New()

		// WHEN
		returnedResolver := resolver.MustRegister(NewTestService)

		// THEN
		assert.Same(t, resolver, returnedResolver)

		service, err := Resolve[*TestService](resolver)
		require.NoError(t, err)
		assert.NotNil(t, service)
	})

	t.Run("it should panic when provider registration fails", func(t *testing.T) {
		// GIVEN
		resolver := New()

		// WHEN & THEN
		assert.Panics(t, func() {
			resolver.MustRegister(func() {
				// not a valid provider function
			})
		})
	})
}
