package option

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test structures for option pattern testing
type ServerConfig struct {
	Host    string
	Port    int
	Timeout int
	Debug   bool
}

func WithHost(host string) Option[ServerConfig] {
	return func(opts *ServerConfig) {
		opts.Host = host
	}
}

func WithPort(port int) Option[ServerConfig] {
	return func(opts *ServerConfig) {
		opts.Port = port
	}
}

func WithTimeout(timeout int) Option[ServerConfig] {
	return func(opts *ServerConfig) {
		opts.Timeout = timeout
	}
}

func WithDebug(debug bool) Option[ServerConfig] {
	return func(opts *ServerConfig) {
		opts.Debug = debug
	}
}

func TestBuild(t *testing.T) {
	t.Run("it should apply single option", func(t *testing.T) {
		// GIVEN
		defaultConfig := &ServerConfig{
			Host:    "localhost",
			Port:    8080,
			Timeout: 30,
			Debug:   false,
		}

		// WHEN
		result := Build(defaultConfig, WithHost("example.com"))

		// THEN
		assert.Equal(t, "example.com", result.Host)
		assert.Equal(t, 8080, result.Port)
		assert.Equal(t, 30, result.Timeout)
		assert.Equal(t, false, result.Debug)
	})

	t.Run("it should apply multiple options", func(t *testing.T) {
		// GIVEN
		defaultConfig := &ServerConfig{
			Host:    "localhost",
			Port:    8080,
			Timeout: 30,
			Debug:   false,
		}

		// WHEN
		result := Build(defaultConfig,
			WithHost("example.com"),
			WithPort(9090),
			WithTimeout(60),
			WithDebug(true),
		)

		// THEN
		assert.Equal(t, "example.com", result.Host)
		assert.Equal(t, 9090, result.Port)
		assert.Equal(t, 60, result.Timeout)
		assert.Equal(t, true, result.Debug)
	})

	t.Run("it should handle no options", func(t *testing.T) {
		// GIVEN
		defaultConfig := &ServerConfig{
			Host:    "localhost",
			Port:    8080,
			Timeout: 30,
			Debug:   false,
		}

		// WHEN
		result := Build(defaultConfig)

		// THEN
		assert.Equal(t, defaultConfig, result)
		assert.Equal(t, "localhost", result.Host)
		assert.Equal(t, 8080, result.Port)
		assert.Equal(t, 30, result.Timeout)
		assert.Equal(t, false, result.Debug)
	})

	t.Run("it should handle overriding options", func(t *testing.T) {
		// GIVEN
		defaultConfig := &ServerConfig{
			Host:    "localhost",
			Port:    8080,
			Timeout: 30,
			Debug:   false,
		}

		// WHEN (applying same option twice, last one wins)
		result := Build(defaultConfig,
			WithPort(9090),
			WithPort(3000),
		)

		// THEN
		assert.Equal(t, "localhost", result.Host)
		assert.Equal(t, 3000, result.Port) // Last option wins
		assert.Equal(t, 30, result.Timeout)
		assert.Equal(t, false, result.Debug)
	})

	t.Run("it should work with different types", func(t *testing.T) {
		// GIVEN
		type DatabaseConfig struct {
			URL         string
			MaxPoolSize int
		}

		defaultDB := &DatabaseConfig{
			URL:         "localhost:5432",
			MaxPoolSize: 10,
		}

		withURL := func(url string) Option[DatabaseConfig] {
			return func(opts *DatabaseConfig) {
				opts.URL = url
			}
		}

		withMaxPoolSize := func(size int) Option[DatabaseConfig] {
			return func(opts *DatabaseConfig) {
				opts.MaxPoolSize = size
			}
		}

		// WHEN
		result := Build(defaultDB,
			withURL("prod.example.com:5432"),
			withMaxPoolSize(50),
		)

		// THEN
		assert.Equal(t, "prod.example.com:5432", result.URL)
		assert.Equal(t, 50, result.MaxPoolSize)
	})
}
