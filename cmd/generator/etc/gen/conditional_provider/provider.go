package services

// @provider named="cache"
// @when named="ENABLE_CACHE" equals="true"
// @when named="ENV" not_equals="test"
// RedisCache provides Redis-based caching
func NewRedisCache() *RedisCache {
	return &RedisCache{}
}

type RedisCache struct{}

// @provider named="cache"
// MemoryCache provides in-memory caching
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{}
}

type MemoryCache struct{}
