package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/OpenListTeam/go-cache"
	"github.com/redis/go-redis/v9"
)

type RedisCache[V any] struct {
	client *redis.Client
	prefix string
}

func NewRedisCache[V any](client *redis.Client, prefix string) cache.ICache[V] {
	return &RedisCache[V]{
		client: client,
		prefix: prefix,
	}
}

type redisItem struct {
	expire time.Time
}

func (ri *redisItem) Expired() bool {
	return !ri.expire.IsZero() && time.Now().After(ri.expire)
}

func (ri *redisItem) CanExpire() bool {
	return !ri.expire.IsZero()
}

func (ri *redisItem) SetExpireAt(t time.Time) {
	ri.expire = t
}

func (r *RedisCache[V]) Set(k string, v V, opts ...cache.SetIOption[V]) bool {
	ri := &redisItem{}
	for _, opt := range opts {
		if pass := opt(r, k, ri); !pass {
			return false
		}
	}

	var expiration time.Duration
	if !ri.expire.IsZero() {
		expiration = time.Until(ri.expire)
		if expiration <= 0 {
			r.Del(k)
			return true
		}
	}

	data, err := json.Marshal(v)
	if err != nil {
		return false
	}

	err = r.client.Set(context.Background(), r.prefix+k, data, expiration).Err()
	return err == nil
}

func (r *RedisCache[V]) Get(k string) (V, bool) {
	var zero V
	data, err := r.client.Get(context.Background(), r.prefix+k).Bytes()
	if err != nil {
		return zero, false
	}
	var val V
	err = json.Unmarshal(data, &val)
	if err != nil {
		return zero, false
	}
	return val, true
}

func (r *RedisCache[V]) GetSet(k string, v V, opts ...cache.SetIOption[V]) (V, bool) {
	oldVal, found := r.Get(k)
	r.Set(k, v, opts...)
	return oldVal, found
}

func (r *RedisCache[V]) GetDel(k string) (V, bool) {
	val, found := r.Get(k)
	if found {
		r.Del(k)
	}
	return val, found
}

func (r *RedisCache[V]) Del(keys ...string) int {
	if len(keys) == 0 {
		return 0
	}
	prefixedKeys := make([]string, len(keys))
	for i, k := range keys {
		prefixedKeys[i] = r.prefix + k
	}
	count, err := r.client.Del(context.Background(), prefixedKeys...).Result()
	if err != nil {
		return 0
	}
	return int(count)
}

func (r *RedisCache[V]) DelExpired(k string) bool {
	return true
}

func (r *RedisCache[V]) Exists(keys ...string) bool {
	if len(keys) == 0 {
		return true
	}
	prefixedKeys := make([]string, len(keys))
	for i, k := range keys {
		prefixedKeys[i] = r.prefix + k
	}
	count, err := r.client.Exists(context.Background(), prefixedKeys...).Result()
	if err != nil {
		return false
	}
	return count == int64(len(keys))
}

func (r *RedisCache[V]) Expire(k string, d time.Duration) bool {
	success, err := r.client.Expire(context.Background(), r.prefix+k, d).Result()
	return err == nil && success
}

func (r *RedisCache[V]) ExpireAt(k string, t time.Time) bool {
	success, err := r.client.ExpireAt(context.Background(), r.prefix+k, t).Result()
	return err == nil && success
}

func (r *RedisCache[V]) Persist(k string) bool {
	success, err := r.client.Persist(context.Background(), r.prefix+k).Result()
	return err == nil && success
}

func (r *RedisCache[V]) Ttl(k string) (time.Duration, bool) {
	ttl, err := r.client.TTL(context.Background(), r.prefix+k).Result()
	if err != nil || ttl < 0 {
		return 0, false
	}
	return ttl, true
}

func (r *RedisCache[V]) Clear() {
	ctx := context.Background()
	iter := r.client.Scan(ctx, 0, r.prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		r.client.Del(ctx, iter.Val())
	}
}
