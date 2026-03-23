package ratelimit

import (
	"context"
	"fmt"
	"time"

	zratelimit "github.com/alexferl/zerohttp/middleware/ratelimit"
	"github.com/redis/go-redis/v9"
)

// RedisClient is the interface for Redis operations used by RedisStore.
// This interface allows for mocking in tests and accepts:
//   - *redis.Client (single-node Redis)
//   - *redis.ClusterClient (Redis Cluster)
//   - redis.UniversalClient (abstract client for any Redis deployment)
type RedisClient interface {
	ZRemRangeByScore(ctx context.Context, key string, min, max string) *redis.IntCmd
	ZCard(ctx context.Context, key string) *redis.IntCmd
	ZRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd
	ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
}

// RedisStore implements ratelimit.Store using Redis for distributed
// rate limiting across multiple server instances.
type RedisStore struct {
	client    RedisClient
	window    time.Duration
	rate      int
	algorithm zratelimit.Algorithm
	keyPrefix string
}

// NewRedisStore creates a new Redis-backed rate limit store.
// This allows rate limiting to work across multiple server instances.
// The client can be *redis.Client, *redis.ClusterClient, redis.UniversalClient, or any type
// implementing the RedisClient interface.
func NewRedisStore(client RedisClient, algorithm zratelimit.Algorithm, window time.Duration, rate int) *RedisStore {
	return &RedisStore{
		client:    client,
		window:    window,
		rate:      rate,
		algorithm: algorithm,
		keyPrefix: "ratelimit:",
	}
}

// CheckAndRecord implements the Store interface using Redis.
// Uses sliding window algorithm with Redis sorted sets.
func (s *RedisStore) CheckAndRecord(ctx context.Context, key string, now time.Time) (bool, int, time.Time) {
	windowStart := now.Add(-s.window)
	redisKey := s.keyPrefix + key

	// Remove old entries outside the window
	s.client.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))

	// Count current entries in window
	count, err := s.client.ZCard(ctx, redisKey).Result()
	if err != nil {
		// On error, allow the request (fail open)
		return true, s.rate - 1, now.Add(s.window)
	}

	if int(count) >= s.rate {
		// Rate limit exceeded
		oldest, _ := s.client.ZRangeWithScores(ctx, redisKey, 0, 0).Result()
		resetTime := now.Add(s.window)
		if len(oldest) > 0 {
			resetTime = time.UnixMilli(int64(oldest[0].Score)).Add(s.window)
		}
		return false, 0, resetTime
	}

	// Add current request to the window
	err = s.client.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: now.UnixNano(),
	}).Err()
	if err != nil {
		return true, s.rate - 1, now.Add(s.window)
	}

	// Set expiry on the key to auto-cleanup
	s.client.Expire(ctx, redisKey, s.window)

	remaining := s.rate - int(count) - 1
	resetTime := now.Add(s.window)

	return true, remaining, resetTime
}
