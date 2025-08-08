package app

// @config prefix="APP"
// AppConfig contains application configuration
type AppConfig struct {
	DatabaseURL string `env:"DATABASE_URL"`
	LogLevel    string `env:"LOG_LEVEL"`
	Port        int    `env:"PORT"`
}
