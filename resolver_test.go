package godi

import (
	"errors"
	"fmt"
	"github.com/a-peyrard/godi/concurrent"
	"github.com/a-peyrard/godi/slices"
	"io"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strconv"
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
		assert.Contains(t, err.Error(), "no providers found")
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
		assert.Contains(t, err.Error(), "multiple providers found for")
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
		assert.Len(t, resolved, 3) // our 2 services, and the resolver itself!
		types := slices.Map(resolved, func(c io.Closer) string {
			return fmt.Sprintf("%T", c)
		})
		assert.Contains(t, types, "*godi.TestService")
		assert.Contains(t, types, "*godi.TestRepository")
	})

	t.Run("it should handle circular dependencies gracefully", func(t *testing.T) {
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
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dependency cycle detected")
	})

	t.Run("it should handle graph of dependencies", func(t *testing.T) {
		// GIVEN
		//       A
		//      / \
		//     B   C
		//      \ /
		//       D
		resolver := New()

		providerA := func(b string, c string) string {
			return fmt.Sprintf("A depends on B (%s) and C (%s)", b, c)
		}
		providerB := func(d string) string {
			return fmt.Sprintf("B depends on D (%s)", d)
		}
		providerC := func(d string) string {
			return fmt.Sprintf("C depends on D (%s)", d)
		}
		providerD := func() string {
			return "D is independent"
		}

		resolver.MustRegister(providerA, Named("A"), Dependencies(Inject.Named("B"), Inject.Named("C")))
		resolver.MustRegister(providerB, Named("B"), Dependencies(Inject.Named("D")))
		resolver.MustRegister(providerC, Named("C"), Dependencies(Inject.Named("D")))
		resolver.MustRegister(providerD, Named("D"))

		// WHEN
		value, err := ResolveNamed[string](resolver, "A")

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "A depends on B (B depends on D (D is independent)) and C (C depends on D (D is independent))", value)
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

	t.Run("it should not fail if dependency are missing but flagged as optional", func(t *testing.T) {
		// GIVEN
		resolver := New()
		resolver.MustRegister(
			func() string {
				return "bar"
			},
			Named("bar"),
		)
		resolver.MustRegister(
			func(foo string, bar string) string {
				return foo + bar
			},
			Named("foobar"),
			Dependencies(
				Inject.Named("foo").Optional(),
				Inject.Named("bar").Optional(),
			),
		)

		// WHEN
		value, err := ResolveNamed[string](resolver, "foobar")

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "bar", value)
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

	ComplexComponent struct {
		foo         string
		answer      int
		bar         string
		tokens      []string
		namedTokens map[string]string
	}
)

func (n *NameSupplier) Name() string {
	return n.name
}

func TestResolver_Register(t *testing.T) {
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
		assert.Contains(t, err.Error(), "provider must be either a function")
	})

	t.Run("it should fail if function does not return anything", func(t *testing.T) {
		// GIVEN
		resolver := New()

		// WHEN
		err := resolver.Register(func() {
			// no return value
		})

		// THEN
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must either return the instance and an error")
	})

	t.Run("it should fail if function does not return an error as second element", func(t *testing.T) {
		// GIVEN
		resolver := New()

		// WHEN
		err := resolver.Register(func() (string, string) {
			return "really", "not a valid provider"
		})

		// THEN
		require.Error(t, err)
		assert.Contains(t, err.Error(), "returns two elements, it must return an error")
	})

	t.Run("it should fail if function does return more than two elements", func(t *testing.T) {
		// GIVEN
		resolver := New()

		// WHEN
		err := resolver.Register(func() (string, string, error) {
			return "really", "not a valid provider", nil
		})

		// THEN
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must either return the instance and an error")
	})

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

	t.Run("it should allows to register with named dependencies", func(t *testing.T) {
		// GIVEN
		resolver := New()
		resolver.MustRegister(
			func(foo string, answer int, bar string) *ComplexComponent {
				return &ComplexComponent{
					foo:    foo,
					answer: answer,
					bar:    bar,
				}
			},
			Dependencies(
				Inject.Named("myFoo"),
				Inject.Auto(),
				Inject.Named("myBar"),
			),
		)
		resolver.MustRegister(
			func() string {
				return "this is the foo string"
			},
			Named("myFoo"),
		)
		resolver.MustRegister(
			func() string {
				return "this is the bar string"
			},
			Named("myBar"),
		)
		resolver.MustRegister(
			func() int {
				return 42
			},
			Named("answer to everything"),
		)

		// WHEN
		complexComp, err := Resolve[*ComplexComponent](resolver)

		// THEN
		require.NoError(t, err)
		assert.NotNil(t, complexComp)
		assert.Equal(t, "this is the foo string", complexComp.foo)
		assert.Equal(t, 42, complexComp.answer)
		assert.Equal(t, "this is the bar string", complexComp.bar)
	})

	t.Run("it should allows to register with slice as a dependency resolving all", func(t *testing.T) {
		// GIVEN
		resolver := New()
		resolver.MustRegister(
			func(tokens []string) *ComplexComponent {
				return &ComplexComponent{
					tokens: tokens,
				}
			},
			Dependencies(
				Inject.Multiple(),
			),
		)
		resolver.MustRegister(
			func() string {
				return "this is the foo string"
			},
			Named("myFoo"),
		)
		resolver.MustRegister(
			func() string {
				return "this is the bar string"
			},
			Named("myBar"),
		)
		resolver.MustRegister(
			func() int {
				return 42
			},
			Named("answer to everything"),
		)

		// WHEN
		complexComp, err := Resolve[*ComplexComponent](resolver)

		// THEN
		require.NoError(t, err)
		assert.NotNil(t, complexComp)
		assert.Len(t, complexComp.tokens, 2)
		assert.Contains(t, complexComp.tokens, "this is the foo string")
		assert.Contains(t, complexComp.tokens, "this is the bar string")
	})

	t.Run("it should just treat slice as regular dependencies if multiple is not specified", func(t *testing.T) {
		// GIVEN
		resolver := New()
		resolver.MustRegister(
			func(tokens []string) *ComplexComponent {
				return &ComplexComponent{
					tokens: tokens,
				}
			},
		)
		resolver.MustRegister(
			func() string {
				return "this is the foo string"
			},
			Named("myFoo"),
		)
		resolver.MustRegister(
			func() string {
				return "this is the bar string"
			},
			Named("myBar"),
		)
		resolver.MustRegister(
			func() []string {
				return []string{"hello", "Augustin", "how are you?"}
			},
			Named("some strings"),
		)

		// WHEN
		complexComp, err := Resolve[*ComplexComponent](resolver)

		// THEN
		require.NoError(t, err)
		assert.NotNil(t, complexComp)
		assert.Len(t, complexComp.tokens, 3)
		assert.Equal(t, []string{"hello", "Augustin", "how are you?"}, complexComp.tokens)
	})

	t.Run("it should allows to use map as a container for dependencies tagged as multiple", func(t *testing.T) {
		// GIVEN
		resolver := New()
		resolver.MustRegister(
			func(namedTokens map[string]string) *ComplexComponent {
				return &ComplexComponent{
					namedTokens: namedTokens,
				}
			},
			Dependencies(
				Inject.Multiple(),
			),
		)
		resolver.MustRegister(
			func() string {
				return "this is the foo string"
			},
			Named("myFoo"),
		)
		resolver.MustRegister(
			func() string {
				return "this is the bar string"
			},
			Named("myBar"),
		)
		resolver.MustRegister(
			func() int {
				return 42
			},
			Named("answer to everything"),
		)

		// WHEN
		complexComp, err := Resolve[*ComplexComponent](resolver)

		// THEN
		require.NoError(t, err)
		assert.NotNil(t, complexComp)
		assert.Len(t, complexComp.namedTokens, 2)
		assert.Equal(t, "this is the foo string", complexComp.namedTokens["myFoo"])
		assert.Equal(t, "this is the bar string", complexComp.namedTokens["myBar"])
	})

	t.Run("it should handle map as regular components if not tagged as multiple", func(t *testing.T) {
		// GIVEN
		resolver := New()
		resolver.MustRegister(
			func(namedTokens map[string]string) *ComplexComponent {
				return &ComplexComponent{
					namedTokens: namedTokens,
				}
			},
		)
		resolver.MustRegister(
			func() string {
				return "this is the foo string"
			},
			Named("myFoo"),
		)
		resolver.MustRegister(
			func() string {
				return "this is the bar string"
			},
			Named("myBar"),
		)
		resolver.MustRegister(
			func() map[string]string {
				return map[string]string{
					"foo":   "bar",
					"hello": "world",
				}
			},
			Named("answer to everything"),
		)

		// WHEN
		complexComp, err := Resolve[*ComplexComponent](resolver)

		// THEN
		require.NoError(t, err)
		assert.NotNil(t, complexComp)
		assert.Len(t, complexComp.namedTokens, 2)
		assert.Equal(t, "bar", complexComp.namedTokens["foo"])
		assert.Equal(t, "world", complexComp.namedTokens["hello"])
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

	t.Run("it should allow to register provider that don't return errors", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(func() *TestService {
			return &TestService{Name: "test-service"}
		})
		require.NoError(t, err)

		// WHEN
		service, err := Resolve[*TestService](resolver)

		// THEN
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Equal(t, "test-service", service.Name)
	})

	t.Run("it should allow to inject resolver into a provider", func(t *testing.T) {
		// GIVEN
		resolver := New()
		resolver.MustRegister(func(r *Resolver) (*TestService, error) {
			dynResolution, err := ResolveNamed[string](r, "str.foo")
			if err != nil {
				return nil, fmt.Errorf("failed to resolve str.foo: %w", err)
			}
			return &TestService{Name: dynResolution}, nil
		})
		resolver.MustRegister(ToStaticProvider("hello world"), Named("str.foo"))
		resolver.MustRegister(ToStaticProvider("waldo"), Named("str.bar"))

		// WHEN
		service, err := Resolve[*TestService](resolver)

		// THEN
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Equal(t, "hello world", service.Name)
	})

	t.Run("it should allow conditional providers, and register if condition is met", func(t *testing.T) {
		// GIVEN
		resolver := New()
		resolver.MustRegister(
			func() string {
				return "My App [PROD MODE]"
			},
			Named("short_description"),
		)
		resolver.MustRegister(
			func() string {
				return "dev"
			},
			Named("APP_ENV"),
		)

		// WHEN
		resolver.MustRegister(
			func() string {
				return "My App [DEV MODE]"
			},
			Named("short_description"),
			Priority(100),
			When("APP_ENV").Equals("dev"),
		)

		// THEN
		val, err := ResolveNamed[string](resolver, "short_description")
		require.NoError(t, err)
		assert.Equal(t, "My App [DEV MODE]", val)
	})

	t.Run("it should allow conditional providers, and not register if condition is not met", func(t *testing.T) {
		// GIVEN
		resolver := New()
		resolver.MustRegister(
			func() string {
				return "My App [PROD MODE]"
			},
			Named("short_description"),
		)
		resolver.MustRegister(
			func() string {
				return "production"
			},
			Named("APP_ENV"),
		)

		// WHEN
		resolver.MustRegister(
			func() string {
				return "My App [DEV MODE]"
			},
			Named("short_description"),
			Priority(100),
			When("APP_ENV").NotEquals("production"),
		)

		// THEN
		val, err := ResolveNamed[string](resolver, "short_description")
		require.NoError(t, err)
		assert.Equal(t, "My App [PROD MODE]", val)
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

type SomeProvider struct {
	known      map[string]string
	buildCount atomic.Int32
}

func (e *SomeProvider) CanProvide(name Name) bool {
	if name.typ == StringType && name.name != "" {
		_, found := e.known[name.name]
		if found {
			return true
		}
	}

	return false
}

func (e *SomeProvider) Provide(n Name, _ []reflect.Value) (comp reflect.Value, err error) {
	e.buildCount.Add(1)
	val, found := e.known[n.name]
	if !found {
		return reflect.Value{}, fmt.Errorf("unknown name: %s", n.name)
	}
	return reflect.ValueOf(val), nil
}

func (e *SomeProvider) Dependencies() []Request {
	return nil
}

func (e *SomeProvider) Priority() int {
	return 0
}

func (e *SomeProvider) ListProvidableNames() []Name {
	names := make([]Name, 0, len(e.known))
	for key := range e.known {
		names = append(names, Name{
			name: key,
			typ:  StringType,
		})
	}
	return names
}

func (e *SomeProvider) Description() string {
	return "some test provider"
}

func TestResolver_Provider(t *testing.T) {
	t.Run("it should register provider and allow to resolve by name", func(t *testing.T) {
		// GIVEN
		resolver := New()
		dynamicProvider := &SomeProvider{
			known: map[string]string{
				"str.foo": "hello world",
				"str.bar": "waldo",
			},
		}

		// WHEN
		resolver.MustRegister(dynamicProvider)

		// THEN
		resolveNamed, err := ResolveNamed[string](resolver, "str.foo")
		require.NoError(t, err)

		assert.Equal(t, "hello world", resolveNamed)
	})

	t.Run("it should build provider only once", func(t *testing.T) {
		// GIVEN
		resolver := New()
		dynamicProvider := &SomeProvider{
			known: map[string]string{
				"str.foo": "hello world",
				"str.bar": "waldo",
			},
		}
		resolver.MustRegister(dynamicProvider)

		// WHEN
		_, err := ResolveNamed[string](resolver, "str.foo")
		require.NoError(t, err)
		_, err = ResolveNamed[string](resolver, "str.foo")
		require.NoError(t, err)
		resolveNamed, err := ResolveNamed[string](resolver, "str.foo")

		// THEN
		assert.Equal(t, "hello world", resolveNamed)
		// only one build, all other calls should use the built provider
		assert.Equal(t, int32(1), dynamicProvider.buildCount.Load())
	})

	t.Run("it should allow to get all from type", func(t *testing.T) {
		// GIVEN
		resolver := New()
		dynamicProvider := &SomeProvider{
			known: map[string]string{
				"str.foo": "hello world",
				"str.bar": "waldo",
			},
		}
		resolver.MustRegister(dynamicProvider)

		// WHEN
		allStr, err := ResolveAll[string](resolver)
		require.NoError(t, err)

		// THEN
		assert.GreaterOrEqual(t, len(allStr), 2)
		assert.Contains(t, allStr, "hello world")
		assert.Contains(t, allStr, "waldo")
	})

	t.Run("it should not produce new types or call build if called multiple times", func(t *testing.T) {
		// GIVEN
		resolver := New()
		dynamicProvider := &SomeProvider{
			known: map[string]string{
				"str.foo": "hello world",
				"str.bar": "waldo",
			},
		}
		resolver.MustRegister(dynamicProvider)

		// WHEN
		allStr, err := ResolveAll[string](resolver)
		require.NoError(t, err)
		originalLength := len(allStr)

		_, err = ResolveAll[string](resolver)
		require.NoError(t, err)
		allStr, err = ResolveAll[string](resolver)
		require.NoError(t, err)

		// THEN
		assert.Equal(t, originalLength, len(allStr))
		// only one build per buildable names (i.e. 2), all other calls should use the built provider
		assert.Equal(t, int32(2), dynamicProvider.buildCount.Load())
	})
}

func TestResolver_ThreadSafe(t *testing.T) {
	t.Run("it should allow concurrent resolutions", func(t *testing.T) {
		// GIVEN
		resolver := New()

		var syncStart sync.WaitGroup
		syncStart.Add(1)
		var syncEnd sync.WaitGroup
		syncEnd.Add(2)

		buildIndex := atomic.Int32{}
		provider := func() string {
			syncStart.Wait()
			val := buildIndex.Add(1)
			return fmt.Sprintf("service-%d", val)
		}
		resolver.MustRegister(provider, Named("myService"))

		// WHEN
		result := make([]string, 2)
		go func() {
			defer syncEnd.Done()
			res, err := ResolveNamed[string](resolver, "myService")
			require.NoError(t, err)
			result[0] = res
		}()
		go func() {
			defer syncEnd.Done()
			res, err := ResolveNamed[string](resolver, "myService")
			require.NoError(t, err)
			result[1] = res
		}()
		time.Sleep(10 * time.Millisecond)
		syncStart.Done() // let the provider build

		// THEN
		syncEnd.Wait()
		assert.Equal(t, "service-1", result[0])
		assert.Equal(t, "service-1", result[1])
	})

	t.Run("it should allow concurrent registers and resolutions", func(t *testing.T) {
		// GIVEN
		ctx, cancelFunc := context.WithCancel(t.Context())

		resolver := New()
		namePrefix := "foobar-"
		target := namePrefix + "5"

		// WHEN

		// one goroutine continuously registering new providers till context is done
		go func() {
			idx := 1
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Millisecond):
					value := namePrefix + strconv.Itoa(idx)
					resolver.MustRegister(ToStaticProvider(value), Named(value))
				}
				idx++
			}
		}()

		foundChan := make(chan string)
		go func() {
			for {
				value, found, err := TryResolveNamed[string](resolver, target)
				if err != nil {
					cancelFunc()
					return
				}
				if found {
					foundChan <- value
					return
				}
			}
		}()

		// THEN
		var foundValue string
		select {
		case foundValue = <-foundChan:
		case <-time.After(2 * time.Second):
		}

		cancelFunc() // stop the register goroutine

		assert.Equal(t, target, foundValue)
	})
}

func TestResolver_Initialize(t *testing.T) {
	t.Run("it should run initializers", func(t *testing.T) {
		// GIVEN
		resolver := New()
		slice := concurrent.NewSlice[string]()
		resolver.MustRegister(func() func() {
			return func() {
				slice.Append("init1")
			}
		})
		resolver.MustRegister(func() func() {
			return func() {
				slice.Append("init2")
			}
		})
		resolver.MustRegister(func() func() error {
			return func() error {
				slice.Append("unsafe init1")
				return nil
			}
		})

		// WHEN
		err := resolver.Initialize()

		// THEN
		require.NoError(t, err)
		values := slice.Get()
		require.Len(t, values, 3)
		assert.Contains(t, values, "init1")
		assert.Contains(t, values, "init2")
		assert.Contains(t, values, "unsafe init1")
	})

	t.Run("it should allow to initialize without catching errors", func(t *testing.T) {
		// GIVEN
		resolver := New()
		slice := concurrent.NewSlice[string]()
		resolver.MustRegister(func() func() {
			return func() {
				slice.Append("init1")
			}
		})
		resolver.MustRegister(func() func() {
			return func() {
				slice.Append("init2")
			}
		})
		resolver.MustRegister(func() func() error {
			return func() error {
				slice.Append("unsafe init1")
				return nil
			}
		})

		// WHEN
		resolver.MustInitialize()

		// THEN
		values := slice.Get()
		require.Len(t, values, 3)
		assert.Contains(t, values, "init1")
		assert.Contains(t, values, "init2")
		assert.Contains(t, values, "unsafe init1")
	})
}

func TestResolver_MustResolve(t *testing.T) {
	t.Run("it should resolve a component", func(t *testing.T) {
		// GIVEN
		resolver := New()
		err := resolver.Register(NewTestService)
		require.NoError(t, err)

		// WHEN
		service1 := MustResolve[*TestService](resolver)
		require.NoError(t, err)

		// THEN
		assert.Equal(t, service1.Name, "test-service")
	})
}
