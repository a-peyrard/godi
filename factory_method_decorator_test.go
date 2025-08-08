package godi

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for decorator testing
type DatabaseService interface {
	Connect() error
	Query(sql string) (string, error)
}

type SimpleDatabaseService struct {
	URL string
}

func (s *SimpleDatabaseService) Connect() error {
	return nil
}

func (s *SimpleDatabaseService) Query(sql string) (string, error) {
	return fmt.Sprintf("result for: %s", sql), nil
}

type LoggingDatabaseService struct {
	wrapped DatabaseService
	logger  *TestLogger
}

func (l *LoggingDatabaseService) Connect() error {
	l.logger.Log("Connecting to database")
	return l.wrapped.Connect()
}

func (l *LoggingDatabaseService) Query(sql string) (string, error) {
	l.logger.Log(fmt.Sprintf("Executing query: %s", sql))
	return l.wrapped.Query(sql)
}

type CachingDatabaseService struct {
	wrapped DatabaseService
	cache   map[string]string
}

func (c *CachingDatabaseService) Connect() error {
	return c.wrapped.Connect()
}

func (c *CachingDatabaseService) Query(sql string) (string, error) {
	if result, exists := c.cache[sql]; exists {
		return fmt.Sprintf("cached: %s", result), nil
	}
	result, err := c.wrapped.Query(sql)
	if err == nil {
		c.cache[sql] = result
	}
	return result, err
}

type Config struct {
	LogLevel string
}

// Decorator factory methods for testing
func AddLoggingDecorator(db DatabaseService, logger *TestLogger) DatabaseService {
	return &LoggingDatabaseService{
		wrapped: db,
		logger:  logger,
	}
}

func AddCachingDecorator(db DatabaseService) DatabaseService {
	return &CachingDatabaseService{
		wrapped: db,
		cache:   make(map[string]string),
	}
}

func AddLoggingWithError(db DatabaseService, logger *TestLogger) (DatabaseService, error) {
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	return &LoggingDatabaseService{
		wrapped: db,
		logger:  logger,
	}, nil
}

func FailingDecorator(db DatabaseService) (DatabaseService, error) {
	return nil, errors.New("decoration failed")
}

func InvalidDecorator() DatabaseService {
	return &SimpleDatabaseService{URL: "invalid"}
}

func WrongTypeDecorator(db DatabaseService) *TestLogger {
	return &TestLogger{Level: "wrong"}
}

func TestFactoryMethodDecorator(t *testing.T) {
	t.Run("it should create decorator from simple factory method", func(t *testing.T) {
		// GIVEN
		factoryMethod := AddCachingDecorator

		// WHEN
		decorator, err := NewFactoryMethodDecorator(factoryMethod, Decorate("foobar"))

		// THEN
		require.NoError(t, err)
		require.NotNil(t, decorator)

		// Verify decorator properties
		forName := decorator.ForName()
		assert.Equal(t, "foobar", forName.name)
		assert.Equal(t, reflect.TypeOf((*DatabaseService)(nil)).Elem(), forName.typ)
		assert.Equal(t, 0, decorator.Priority())  // Default priority
		assert.Empty(t, decorator.Dependencies()) // No additional dependencies (only the component to decorate)
	})

	t.Run("it should create decorator from factory method with dependencies", func(t *testing.T) {
		// GIVEN
		factoryMethod := AddLoggingDecorator

		// WHEN
		decorator, err := NewFactoryMethodDecorator(factoryMethod, Decorate("foobar"))

		// THEN
		require.NoError(t, err)
		require.NotNil(t, decorator)

		// Verify decorator properties
		forName := decorator.ForName()
		assert.Equal(t, "foobar", forName.name)
		assert.Equal(t, reflect.TypeOf((*DatabaseService)(nil)).Elem(), forName.typ)

		deps := decorator.Dependencies()
		require.Len(t, deps, 1) // Only additional dependencies (logger), not the decorated component
	})

	t.Run("it should create decorator with custom options", func(t *testing.T) {
		// GIVEN
		factoryMethod := AddCachingDecorator

		// WHEN
		decorator, err := NewFactoryMethodDecorator(
			factoryMethod,
			Decorate("custom.caching"),
			Priority(200),
			Description("Adds caching functionality"),
		)

		// THEN
		require.NoError(t, err)
		require.NotNil(t, decorator)

		// Verify custom options were applied
		forName := decorator.ForName()
		assert.Equal(t, "custom.caching", forName.name)
		assert.Equal(t, 200, decorator.Priority())
		assert.Equal(t, "Adds caching functionality", decorator.Description())
	})

	t.Run("it should reject non-function factory methods", func(t *testing.T) {
		// GIVEN
		notAFunction := "this is not a function"

		// WHEN
		decorator, err := NewFactoryMethodDecorator(notAFunction, Decorate("foobar"))

		// THEN
		require.Error(t, err)
		assert.Nil(t, decorator)
		assert.Contains(t, err.Error(), "factory method must be a function")
	})

	t.Run("it should reject factory methods with no parameters", func(t *testing.T) {
		// GIVEN
		factoryMethod := InvalidDecorator

		// WHEN
		decorator, err := NewFactoryMethodDecorator(factoryMethod, Decorate("foobar"))

		// THEN
		require.Error(t, err)
		assert.Nil(t, decorator)
		assert.Contains(t, err.Error(), "factory method must have at least one parameter")
	})

	t.Run("it should reject factory methods where return type doesn't match first parameter", func(t *testing.T) {
		// GIVEN
		factoryMethod := WrongTypeDecorator

		// WHEN
		decorator, err := NewFactoryMethodDecorator(factoryMethod, Decorate("foobar"))

		// THEN
		require.Error(t, err)
		assert.Nil(t, decorator)
		assert.Contains(t, err.Error(), "the first parameter of the factory method must be the same type as the return type")
	})

	t.Run("it should decorate component successfully with no dependencies", func(t *testing.T) {
		// GIVEN
		decorator, err := NewFactoryMethodDecorator(AddCachingDecorator, Decorate("foobar"))
		require.NoError(t, err)

		originalDB := &SimpleDatabaseService{URL: "localhost:5432"}

		// WHEN
		decorated, err := decorator.Decorate(reflect.ValueOf(originalDB), []reflect.Value{})

		// THEN
		require.NoError(t, err)
		require.True(t, decorated.IsValid())

		decoratedDB, ok := decorated.Interface().(DatabaseService)
		require.True(t, ok)

		// Verify decoration worked - should be wrapped with caching
		cachingDB, isCaching := decoratedDB.(*CachingDatabaseService)
		require.True(t, isCaching)
		assert.Same(t, originalDB, cachingDB.wrapped)
	})

	t.Run("it should handle decorator method that returns error", func(t *testing.T) {
		// GIVEN
		decorator, err := NewFactoryMethodDecorator(FailingDecorator, Decorate("foobar"))
		require.NoError(t, err)

		originalDB := &SimpleDatabaseService{URL: "localhost:5432"}

		// WHEN
		decorated, err := decorator.Decorate(reflect.ValueOf(originalDB), []reflect.Value{})

		// THEN
		require.Error(t, err)
		assert.False(t, decorated.IsValid())
		assert.Contains(t, err.Error(), "decoration failed")
	})

	t.Run("it should handle decorator method with error validation", func(t *testing.T) {
		// GIVEN
		decorator, err := NewFactoryMethodDecorator(AddLoggingWithError, Decorate("foobar"))
		require.NoError(t, err)

		originalDB := &SimpleDatabaseService{URL: "localhost:5432"}

		// Test with nil logger (should cause error)
		dependencies := []reflect.Value{
			reflect.ValueOf((*TestLogger)(nil)),
		}

		// WHEN
		decorated, err := decorator.Decorate(reflect.ValueOf(originalDB), dependencies)

		// THEN
		require.Error(t, err)
		assert.False(t, decorated.IsValid())
		assert.Contains(t, err.Error(), "logger cannot be nil")
	})

	t.Run("it should handle panic in decorator method gracefully", func(t *testing.T) {
		// GIVEN
		panickyDecorator := func(db DatabaseService) DatabaseService {
			panic("decoration panic")
		}
		decorator, err := NewFactoryMethodDecorator(panickyDecorator, Decorate("foobar"))
		require.NoError(t, err)

		originalDB := &SimpleDatabaseService{URL: "localhost:5432"}

		// WHEN
		decorated, err := decorator.Decorate(reflect.ValueOf(originalDB), []reflect.Value{})

		// THEN
		require.Error(t, err)
		assert.False(t, decorated.IsValid())
		assert.Contains(t, err.Error(), "panic calling provider")
		assert.Contains(t, err.Error(), "decoration panic")
	})

	t.Run("it should provide correct string representation", func(t *testing.T) {
		// GIVEN
		decorator, err := NewFactoryMethodDecorator(AddCachingDecorator, Decorate("foobar"))
		require.NoError(t, err)

		// WHEN
		stringRepr := decorator.(fmt.Stringer).String()

		// THEN
		assert.Contains(t, stringRepr, "FactoryMethodDecorator")
		assert.Contains(t, stringRepr, "AddCachingDecorator")
		assert.Contains(t, stringRepr, "DatabaseService")
	})

	t.Run("it should handle interface-to-concrete type matching", func(t *testing.T) {
		// GIVEN - decorator that returns concrete type but decorates interface
		concreteDecorator := func(db DatabaseService) *LoggingDatabaseService {
			return &LoggingDatabaseService{
				wrapped: db,
				logger:  &TestLogger{Level: "info"},
			}
		}

		// WHEN
		decorator, err := NewFactoryMethodDecorator(concreteDecorator, Decorate("foobar"))

		// THEN
		require.NoError(t, err)
		require.NotNil(t, decorator)

		forName := decorator.ForName()
		assert.Equal(t, reflect.TypeOf((*DatabaseService)(nil)).Elem(), forName.typ)
	})

	t.Run("it should fail if Decorate option is not provided", func(t *testing.T) {
		// WHEN
		_, err := NewFactoryMethodDecorator(AddCachingDecorator)

		// THEN
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no decorate option provided")
	})
}
