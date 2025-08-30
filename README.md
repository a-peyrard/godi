# goDI

A lightweight, annotation-based dependency injection framework for Go that provides compile-time code generation and runtime efficiency.

## Table of Contents

- [Overview](#overview)
- [Core Concepts](#core-concepts)
- [Getting Started](#getting-started)
- [Annotations Reference](#annotations-reference)
- [Code Generation](#code-generation)
- [Advanced Features](#advanced-features)
- [Examples](#examples)
- [Best Practices](#best-practices)

## Overview

This DI framework provides:

- **Annotation-based dependency injection** using Go comments
- **Compile-time code generation** for type safety and performance
- **Provider and decorator patterns** for flexible component composition
- **Named dependencies** for disambiguation
- **Priority-based ordering** for deterministic initialization
- **Conditional registration** based on environment or configuration
- **Lifecycle management** with initialization and cleanup hooks

## Core Concepts

### Resolver

The `Resolver` is the central container that manages all dependencies. It's responsible for:

- Registering providers and decorators
- Resolving dependencies at runtime
- Managing component lifecycle
- Ensuring thread-safe operations

```go
resolver := godi.New()
defer resolver.Close()

// Key resolver methods:
// - resolver.Register() - Register providers manually
// - resolver.MustRegister() - Register providers (panics on error)  
// - godi.Resolve[T]() - Resolve dependency (returns T, error)
// - godi.MustResolve[T]() - Resolve dependency (panics on error)
```

### Providers

Providers are functions that create and return instances of dependencies. They can be registered in two ways:

1. **Factory functions** - Annotated with `@provider`
2. **Provider implementations** - Implementing the `Provider` interface

### Decorators

Decorators enhance existing dependencies without modifying their original implementation. They wrap existing components to add cross-cutting concerns like logging, metrics, or validation.

### Named Dependencies

Dependencies can be named to resolve ambiguity when multiple implementations of the same type exist:

```go
resolver.MustRegister(myProvider, godi.Named("database.primary"))
```

## Getting Started

### 1. Set Up Code Generation

Create a registry file that will contain generated registration code:

```go
package registry

import "github.com/a-peyrard/godi"

//go:generate go run github.com/a-peyrard/godi/cmd/generator
type Registry struct {
    godi.EmptyRegistry
}
```

### 2. Initialize the Resolver

```go
func main() {
    resolver := godi.New()
    defer resolver.Close()
    
    // Register built-in providers
    resolver.MustRegister(&godi.EnvProvider{})
    
    // Register auto-discovered providers
    registry.Registry{}.Register(resolver)
    
    // Resolve dependencies
    logger := godi.MustResolve[*zerolog.Logger](resolver)
}
```

### 3. Create Your First Provider

```go
// GetGlobalLogLevel provides the global log level based on configuration.
//
// @provider named="logger.GlobalLogLevel"
func GetGlobalLogLevel(
    logLevel string, // @inject named="Config.LogLevel"
) zerolog.Level {
    level, err := zerolog.ParseLevel(logLevel)
    if err != nil {
        level = zerolog.InfoLevel
    }
    return level
}
```

### 4. Generate Registration Code

Run `go generate` to create the registration code:

```bash
go generate ./...
```

## Annotations Reference

### @provider

Marks a function as a dependency provider.

**Syntax:**
```go
// @provider [named="name"] [priority=number] [description="text"]
```

**Parameters:**
- `named` - Optional name for the dependency
- `priority` - Optional priority (higher numbers = higher priority)
- `description` - Optional description for documentation

**Example:**
```go
// NewDatabase creates a new database connection.
//
// @provider named="database.primary" priority=100
func NewDatabase(
    host string, // @inject named="DB_HOST"
    port int,    // @inject named="DB_PORT"
) *sql.DB {
    // implementation
}
```

### @decorator

Marks a function as a decorator for an existing dependency.

**Syntax:**
```go
// @decorator named="target_dependency_name" [priority=number]
```

**Example:**
```go
// ObservableContextDecorator adds logging to the context.
//
// @decorator named="main.context"
func ObservableContextDecorator(
    toDecorate context.Context,
    writer io.Writer, // @inject named="observability.log_writer"
) context.Context {
    // enhancement logic
    return enhancedContext
}
```

### @inject

Marks a parameter for dependency injection.

**Syntax:**
```go
paramName type, // @inject [named="name"] [optional=true]
```

**Parameters:**
- `named` - Name of the dependency to inject
- `optional=true` - Makes the dependency optional (won't fail if not found)

**Example:**
```go
func NewService(
    db *sql.DB,           // @inject named="database.primary"
    cache redis.Client,   // @inject named="cache" optional=true
    config *Config,       // @inject (injects by type)
) *Service {
    // implementation
}
```

### @config

Marks a struct as a configuration object to be populated from environment variables.

**Syntax:**
```go
// @config [prefix="PREFIX_"]
type Config struct {
    // fields
}
```

### @when

Provides conditional registration based on environment variables.

**Syntax:**
```go
// @when named="ENV_VAR_NAME" equals="value"
```

**Example:**
```go
// NewDevLogger creates a development logger.
//
// @provider named="logger"
// @when named="APP_ENV" equals="dev"
func NewDevLogger() *zerolog.Logger {
    return &zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr})
}
```

## Code Generation

The framework includes a code generator that scans your codebase for annotations and generates registration code.

### How It Works

1. **Scanning**: The generator parses Go source files looking for annotated functions
2. **Analysis**: It extracts dependency information from annotations and function signatures
3. **Generation**: It creates registration code that calls `resolver.MustRegister()` with appropriate options

### Generator Command

The generator is located in `cmd/generator` and can be invoked via `go generate`:

```go
//go:generate go run github.com/a-peyrard/godi/cmd/generator
```

### Generated Output

For a provider like this:

```go
// NewLogWriter provides a log writer.
//
// @provider named="observability.log_writer"
func NewLogWriter(
    appEnv string, // @inject named="APP_ENV"
    newRelicApp *newrelic.Application,
) io.Writer {
    // implementation
}
```

The generator creates:

```go
// Code generated by go generate; DO NOT EDIT!

func (Registry) Register(resolver  godi.Resolver) {
    resolver.MustRegister(
        observability.NewLogWriter,
        godi.Named("observability.log_writer"),
        godi.Description("NewLogWriter provides a log writer."),
        godi.Dependencies(
            godi.Inject.Named("APP_ENV"),
            godi.Inject.Type[*newrelic.Application](),
        ),
    )
}
```

## Advanced Features

### Priority System

Use priorities to control the order of provider execution:

```go
// @provider named="main.logger" priority=100
func NewLogger() *zerolog.Logger {
    // This will override lower priority loggers
}
```

### Conditional Registration

Register components only when certain conditions are met:

```go
// @provider named="cache"
// @when named="REDIS_ENABLED" equals="true"
func NewRedisCache() Cache {
    // Only registered when REDIS_ENABLED=true
}
```

### Lifecycle Management

#### Initialization

Components can implement initialization logic:

```go
// @provider
func DatabaseInitializer(db *sql.DB) func() {
    return func() {
        // Run migrations, create tables, etc.
    }
}
```

#### Cleanup

Components can implement cleanup logic:

```go
// @provider
func DatabaseCloser(db *sql.DB) godi.Closeable {
    return godi.CloseableFunc(func() error {
        return db.Close()
    })
}
```

### Environment-based Configuration

Use the built-in `EnvProvider` to inject environment variables:

```go
resolver.MustRegister(&godi.EnvProvider{})

// Then inject environment variables
func NewService(
    host string, // @inject named="DB_HOST"
    port int,    // @inject named="DB_PORT"
) *Service {
    // implementation
}
```

## Examples

### Complete Example: HTTP Server with Dependencies

```go
// Configuration
type ServerConfig struct {
    Port int    `env:"SERVER_PORT" default:"8080"`
    Host string `env:"SERVER_HOST" default:"localhost"`
}

// @config prefix="SERVER_"
type Config struct {
    Server ServerConfig
}

// Database provider
// @provider named="database"
func NewDatabase(
    host string, // @inject named="DB_HOST"
    port int,    // @inject named="DB_PORT"
) *sql.DB {
    // database connection logic
}

// Service provider
// @provider named="user.service"
func NewUserService(
    db *sql.DB, // @inject named="database"
) *UserService {
    return &UserService{db: db}
}

// HTTP handler provider
// @provider named="user.handler"
func NewUserHandler(
    service *UserService, // @inject named="user.service"
    logger *zerolog.Logger, // @inject named="main.logger"
) *UserHandler {
    return &UserHandler{service: service, logger: logger}
}

// Server provider
// @provider
func NewHTTPServer(
    config *ServerConfig, // @inject
    handler *UserHandler, // @inject named="user.handler"
) *http.Server {
    mux := http.NewServeMux()
    mux.Handle("/users", handler)
    
    return &http.Server{
        Addr:    fmt.Sprintf("%s:%d", config.Host, config.Port),
        Handler: mux,
    }
}

// Logging decorator
// @decorator named="user.service"
func UserServiceLoggingDecorator(
    service *UserService,
    logger *zerolog.Logger, // @inject named="main.logger"
) *UserService {
    return &LoggingUserService{
        UserService: service,
        logger:      logger,
    }
}
```

## Best Practices

### 1. Use Descriptive Names

```go
// Good
// @provider named="database.primary"
// @provider named="cache.redis"

// Avoid
// @provider named="db"
// @provider named="cache"
```

### 2. Prefer Constructor Functions

```go
// Good
// @provider named="user.service"
func NewUserService(deps...) *UserService {
    return &UserService{...}
}

// Avoid complex initialization in providers
```

### 3. Use Decorators for Cross-Cutting Concerns

```go
// Good - Add logging via decorator
// @decorator named="user.service"
func LoggingDecorator(service UserService, logger Logger) UserService

// Avoid - Mixing concerns in the main implementation
```

### 4. Handle Errors Appropriately

```go
// For critical dependencies - panics if not found
db := godi.MustResolve[*sql.DB](resolver)

// For optional dependencies - returns (value, error)
cache, err := godi.Resolve[Cache](resolver)
if err == nil {
    // use cache
}
```

### 5. Group Related Providers

Organize providers by domain or layer:

```bash
internal/
├── database/
│   └── providers.go  # @provider named="database.*"
├── cache/
│   └── providers.go  # @provider named="cache.*"
└── observability/
    └── providers.go  # @provider named="logger.*", "metrics.*"
```

### 6. Use Environment-Specific Providers

```go
// @provider named="logger"
// @when named="APP_ENV" equals="dev"
func NewDevLogger() Logger

// @provider named="logger"
// @when named="APP_ENV" equals="prod"
func NewProdLogger() Logger
```

This documentation provides a comprehensive guide to using the DI framework. The annotation-based approach combined with code generation ensures type safety while maintaining the flexibility of dependency injection.
