package godi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvidesConfig(t *testing.T) {
	t.Run("it should provide simple string value from config", func(t *testing.T) {
		// GIVEN
		type AppConfig struct {
			Name string
		}
		type Config struct {
			App AppConfig
		}
		cfg := Config{
			App: AppConfig{Name: "test-app"},
		}
		provider := ProvidesConfig[Config, string]("App.Name")

		// WHEN
		result, err := provider(cfg)

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "test-app", result)
	})

	t.Run("it should provide integer value from config", func(t *testing.T) {
		// GIVEN
		type DatabaseConfig struct {
			Port int
		}
		type Config struct {
			Database DatabaseConfig
		}
		cfg := Config{
			Database: DatabaseConfig{Port: 5432},
		}
		provider := ProvidesConfig[Config, int]("Database.Port")

		// WHEN
		result, err := provider(cfg)

		// THEN
		require.NoError(t, err)
		assert.Equal(t, 5432, result)
	})

	t.Run("it should provide boolean value from config", func(t *testing.T) {
		// GIVEN
		type DatabaseConfig struct {
			SSL bool
		}
		type Config struct {
			Database DatabaseConfig
		}
		cfg := Config{
			Database: DatabaseConfig{SSL: true},
		}
		provider := ProvidesConfig[Config, bool]("Database.SSL")

		// WHEN
		result, err := provider(cfg)

		// THEN
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("it should provide struct from config", func(t *testing.T) {
		// GIVEN
		type DatabaseConfig struct {
			Host string
			Port int
			SSL  bool
		}
		type Config struct {
			Database DatabaseConfig
		}
		cfg := Config{
			Database: DatabaseConfig{
				Host: "localhost",
				Port: 5432,
				SSL:  true,
			},
		}
		provider := ProvidesConfig[Config, DatabaseConfig]("Database")

		// WHEN
		result, err := provider(cfg)

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "localhost", result.Host)
		assert.Equal(t, 5432, result.Port)
		assert.Equal(t, true, result.SSL)
	})

	t.Run("it should provide pointer to struct from config", func(t *testing.T) {
		// GIVEN
		type AppConfig struct {
			Name    string
			Version string
		}
		type Config struct {
			App *AppConfig
		}
		cfg := Config{
			App: &AppConfig{
				Name:    "test-app",
				Version: "1.0.0",
			},
		}
		provider := ProvidesConfig[Config, *AppConfig]("App")

		// WHEN
		result, err := provider(cfg)

		// THEN
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test-app", result.Name)
		assert.Equal(t, "1.0.0", result.Version)
	})

	t.Run("it should return error for non-existent field", func(t *testing.T) {
		// GIVEN
		type Config struct {
			App string
		}
		cfg := Config{App: "test"}
		provider := ProvidesConfig[Config, string]("NonExistent")

		// WHEN
		_, err := provider(cfg)

		// THEN
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Unable to get value from config")
	})

	t.Run("it should return error for type mismatch", func(t *testing.T) {
		// GIVEN
		type Config struct {
			Value string
		}
		cfg := Config{Value: "not-a-number"}
		provider := ProvidesConfig[Config, int]("Value")

		// WHEN
		_, err := provider(cfg)

		// THEN
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config value at Value is not of type")
	})
}

func TestProvidesConfigWithNestedStructs(t *testing.T) {
	t.Run("it should provide deeply nested configuration", func(t *testing.T) {
		// GIVEN
		type HTTPConfig struct {
			Host string
			Port int
		}

		type GRPCConfig struct {
			Host string
			Port int
		}

		type TLSConfig struct {
			Enabled  bool
			CertFile string
		}

		type ServerConfig struct {
			HTTP HTTPConfig
			GRPC GRPCConfig
			TLS  TLSConfig
		}

		type Config struct {
			Server ServerConfig
		}

		cfg := Config{
			Server: ServerConfig{
				HTTP: HTTPConfig{Host: "0.0.0.0", Port: 8080},
				GRPC: GRPCConfig{Host: "0.0.0.0", Port: 9090},
				TLS:  TLSConfig{Enabled: true, CertFile: "/path/to/cert"},
			},
		}

		provider := ProvidesConfig[Config, ServerConfig]("Server")

		// WHEN
		result, err := provider(cfg)

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "0.0.0.0", result.HTTP.Host)
		assert.Equal(t, 8080, result.HTTP.Port)
		assert.Equal(t, "0.0.0.0", result.GRPC.Host)
		assert.Equal(t, 9090, result.GRPC.Port)
		assert.Equal(t, true, result.TLS.Enabled)
		assert.Equal(t, "/path/to/cert", result.TLS.CertFile)
	})

	t.Run("it should provide nested field values", func(t *testing.T) {
		// GIVEN
		type HTTPConfig struct {
			Host string
			Port int
		}

		type ServerConfig struct {
			HTTP HTTPConfig
		}

		type Config struct {
			Server ServerConfig
		}

		cfg := Config{
			Server: ServerConfig{
				HTTP: HTTPConfig{Host: "localhost", Port: 8080},
			},
		}

		hostProvider := ProvidesConfig[Config, string]("Server.HTTP.Host")
		portProvider := ProvidesConfig[Config, int]("Server.HTTP.Port")

		// WHEN
		host, hostErr := hostProvider(cfg)
		port, portErr := portProvider(cfg)

		// THEN
		require.NoError(t, hostErr)
		require.NoError(t, portErr)
		assert.Equal(t, "localhost", host)
		assert.Equal(t, 8080, port)
	})
}

func TestProvidesConfigWithSlices(t *testing.T) {
	t.Run("it should provide slice of strings", func(t *testing.T) {
		// GIVEN
		type Config struct {
			Servers []string
		}
		cfg := Config{
			Servers: []string{"server1", "server2", "server3"},
		}
		provider := ProvidesConfig[Config, []string]("Servers")

		// WHEN
		result, err := provider(cfg)

		// THEN
		require.NoError(t, err)
		assert.Equal(t, []string{"server1", "server2", "server3"}, result)
	})

	t.Run("it should provide slice of integers", func(t *testing.T) {
		// GIVEN
		type Config struct {
			Ports []int
		}
		cfg := Config{
			Ports: []int{8080, 8081, 8082},
		}
		provider := ProvidesConfig[Config, []int]("Ports")

		// WHEN
		result, err := provider(cfg)

		// THEN
		require.NoError(t, err)
		assert.Equal(t, []int{8080, 8081, 8082}, result)
	})
}

func TestProvidesConfigWithMaps(t *testing.T) {
	t.Run("it should provide map configuration", func(t *testing.T) {
		// GIVEN
		type Config struct {
			Labels map[string]string
		}
		cfg := Config{
			Labels: map[string]string{
				"env":     "production",
				"team":    "backend",
				"version": "1.2.3",
			},
		}

		provider := ProvidesConfig[Config, map[string]string]("Labels")

		// WHEN
		result, err := provider(cfg)

		// THEN
		require.NoError(t, err)
		expected := map[string]string{
			"env":     "production",
			"team":    "backend",
			"version": "1.2.3",
		}
		assert.Equal(t, expected, result)
	})
}

func TestProvidesConfigDirectUsage(t *testing.T) {
	t.Run("it should work when used directly without ProvidesConfig helper", func(t *testing.T) {
		// GIVEN
		type Config struct {
			Value string
		}
		cfg := Config{Value: "hello"}

		var provider ConfigProvider[Config, string] = func(c Config) (string, error) {
			return c.Value, nil
		}

		// WHEN
		result, err := provider(cfg)

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})
}

func TestProvidesConfigErrorHandling(t *testing.T) {
	t.Run("it should handle missing fields gracefully", func(t *testing.T) {
		// GIVEN
		type Config struct {
			Existing string
		}
		cfg := Config{Existing: "value"}
		provider := ProvidesConfig[Config, string]("Missing")

		// WHEN
		_, err := provider(cfg)

		// THEN
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Unable to get value from config")
	})

	t.Run("it should handle type conversion errors", func(t *testing.T) {
		// GIVEN
		type ComplexStruct struct {
			Field1 string
			Field2 int
		}

		type Config struct {
			Simple string
		}

		cfg := Config{Simple: "just-a-string"}
		provider := ProvidesConfig[Config, ComplexStruct]("Simple")

		// WHEN
		_, err := provider(cfg)

		// THEN
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config value at Simple is not of type")
	})
}
