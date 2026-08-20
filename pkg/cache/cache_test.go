package cache

import (
	"testing"

	"github.com/OpenListTeam/go-cache"
)

func TestRedisCacheInterface(t *testing.T) {
	// Compile-time interface verification
	var _ cache.ICache[string] = (*RedisCache[string])(nil)
	var _ cache.ICache[int] = (*RedisCache[int])(nil)
	var _ cache.ICache[any] = (*RedisCache[any])(nil)
}
