package config

// @config prefix="APP"
// AppConfig contains all application settings
type AppConfig struct {
	DatabaseURL   string `env:"DATABASE_URL"`
	RedisURL      string `env:"REDIS_URL"`
	LogLevel      string `env:"LOG_LEVEL"`
	MaxWorkers    int    `env:"MAX_WORKERS"`
	EnableMetrics bool   `env:"ENABLE_METRICS"`
}
