package providers

import (
	"github.com/test/complex/config"
)

// @provider named="app.service" priority=10
// AppService is the main application service
func NewAppService(
	cfg *config.AppConfig, // @inject named="AppConfig"
	cache Cache, // @inject named="cache"
	runners []Runner, // @inject multiple=true
) *AppService {
	return &AppService{}
}

// @provider named="cache"
// @when named="REDIS_ENABLED" equals="true"
// RedisCache for production
func NewRedisCache(cfg *config.AppConfig) Cache { // @inject named="AppConfig"
	return &redisCache{}
}

// @provider named="cache"
// MemCache for development
func NewMemCache() Cache {
	return &memCache{}
}

// @provider named="runner"
// FirstRunner implementation
func NewFirstRunner() Runner {
	return &firstRunner{}
}

// @provider named="runner" priority=10
// SecondRunner implementation
func NewSecondRunner() Runner {
	return &secondRunner{}
}

type AppService struct{}
type Cache interface{}
type Runner interface{}
type redisCache struct{}
type memCache struct{}
type firstRunner struct{}
type secondRunner struct{}
