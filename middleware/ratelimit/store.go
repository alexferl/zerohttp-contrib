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
	Incr(ctx context.Context, key string) *redis.IntCmd
	ExpireNX(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	Close() error
}

// tokenBucketScript is a Lua script that atomically handles token bucket rate limiting.
// Keys: [bucket_key]
// Args: [capacity, refill_rate_per_second, now_unix_millis]
// Returns: [allowed (1 or 0), remaining_tokens, reset_time_unix_millis]
const tokenBucketScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

if tokens == nil then
    tokens = capacity - 1
    redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
    redis.call('EXPIRE', key, math.ceil(capacity / refill_rate))
    return {1, tokens, math.floor(now + (1 / refill_rate) * 1000)}
end

local elapsed = (now - last_refill) / 1000.0
local refill = elapsed * refill_rate
tokens = math.min(capacity, tokens + refill)

if tokens >= 1 then
    tokens = tokens - 1
    redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
    local reset_time = now + math.ceil((1 - tokens) / refill_rate * 1000)
    return {1, math.floor(tokens), reset_time}
else
    redis.call('HSET', key, 'last_refill', now)
    local reset_time = now + math.ceil((1 - tokens) / refill_rate * 1000)
    return {0, 0, reset_time}
end
`

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
// Dispatches to the appropriate algorithm implementation.
func (s *RedisStore) CheckAndRecord(ctx context.Context, key string, now time.Time) (bool, int, time.Time) {
	switch s.algorithm {
	case zratelimit.TokenBucket:
		return s.checkTokenBucket(ctx, key, now)
	case zratelimit.FixedWindow:
		return s.checkFixedWindow(ctx, key, now)
	case zratelimit.SlidingWindow:
		return s.checkSlidingWindow(ctx, key, now)
	default:
		return s.checkTokenBucket(ctx, key, now)
	}
}

// checkTokenBucket implements the token bucket algorithm using a Lua script
// for atomic operations. Supports burst by accumulating unused tokens.
func (s *RedisStore) checkTokenBucket(ctx context.Context, key string, now time.Time) (bool, int, time.Time) {
	redisKey := s.keyPrefix + "tb:" + key
	capacity := float64(s.rate)
	refillRate := capacity / s.window.Seconds() // tokens per second

	result, err := s.client.Eval(ctx, tokenBucketScript, []string{redisKey}, capacity, refillRate, now.UnixMilli()).Result()
	if err != nil {
		// On error, allow the request (fail open)
		return true, s.rate - 1, now.Add(s.window)
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 3 {
		return true, s.rate - 1, now.Add(s.window)
	}

	allowed := values[0].(int64) == 1
	remaining := int(values[1].(int64))
	resetTime := time.UnixMilli(values[2].(int64))

	return allowed, remaining, resetTime
}

// checkFixedWindow implements the fixed window counter algorithm.
// Uses INCR with EXPIRE for atomic counter operations.
func (s *RedisStore) checkFixedWindow(ctx context.Context, key string, now time.Time) (bool, int, time.Time) {
	redisKey := s.keyPrefix + "fw:" + key

	// Increment the counter
	count, err := s.client.Incr(ctx, redisKey).Result()
	if err != nil {
		// On error, allow the request (fail open)
		return true, s.rate - 1, now.Add(s.window)
	}

	// Set expiry on first request in window
	if count == 1 {
		s.client.ExpireNX(ctx, redisKey, s.window)
	}

	// Calculate reset time (end of current window)
	resetTime := now.Add(s.window)

	if int(count) > s.rate {
		return false, 0, resetTime
	}

	remaining := s.rate - int(count)
	return true, remaining, resetTime
}

// checkSlidingWindow implements the sliding window algorithm using Redis sorted sets.
// Tracks individual request timestamps for precise rate limiting.
func (s *RedisStore) checkSlidingWindow(ctx context.Context, key string, now time.Time) (bool, int, time.Time) {
	windowStart := now.Add(-s.window)
	redisKey := s.keyPrefix + "sw:" + key

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

// Close closes the Redis connection.
// Returns an error if the close operation fails.
func (s *RedisStore) Close() error {
	return s.client.Close()
}
