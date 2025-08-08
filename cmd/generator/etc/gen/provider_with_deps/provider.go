package app

import "context"

// @provider named="database.connection" priority=10
// DatabaseConnection provides database connectivity
func NewDatabaseConnection(
	ctx context.Context,
	config *Config, // @inject named="app.config"
	logger Logger, // @inject named="logger" optional=true
) (*DatabaseConnection, error) {
	return &DatabaseConnection{}, nil
}

type DatabaseConnection struct{}
type Config struct{}
type Logger interface{}
