package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/redis/go-redis/v9"
)

// RedisClient is the interface for Redis operations used by RedisStore.
// This interface allows for mocking in tests and accepts:
//   - *redis.Client (single-node Redis)
//   - *redis.ClusterClient (Redis Cluster)
//   - redis.UniversalClient (abstract client for any Redis deployment)
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
	SetNX(ctx context.Context, key string, value any, expiration time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// RedisStore implements config.IdempotencyStore using Redis for distributed
// idempotency across multiple server instances.
type RedisStore struct {
	client    RedisClient
	keyPrefix string
	lockTTL   time.Duration
}

// redisRecord is a JSON-serializable representation of config.IdempotencyRecord.
type redisRecord struct {
	StatusCode int       `json:"status_code"`
	Headers    []string  `json:"headers"`
	Body       []byte    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewRedisStore creates a new Redis-backed idempotency store.
// This allows idempotency to work across multiple server instances.
// The client can be *redis.Client, *redis.ClusterClient, redis.UniversalClient, or any type
// implementing the RedisClient interface.
// The optional prefix is prepended to all keys.
func NewRedisStore(client RedisClient, prefix string) *RedisStore {
	return &RedisStore{
		client:    client,
		keyPrefix: prefix,
		lockTTL:   30 * time.Second, // Default lock TTL
	}
}

// NewRedisStoreWithLockTTL creates a new Redis-backed idempotency store with custom lock TTL.
// The lockTTL determines how long a lock is held before expiring (to prevent deadlocks).
func NewRedisStoreWithLockTTL(client RedisClient, prefix string, lockTTL time.Duration) *RedisStore {
	return &RedisStore{
		client:    client,
		keyPrefix: prefix,
		lockTTL:   lockTTL,
	}
}

// makeKey creates a Redis key with optional prefix.
func (s *RedisStore) makeKey(key string) string {
	if s.keyPrefix != "" {
		return s.keyPrefix + ":" + key
	}
	return key
}

// makeLockKey creates the lock key for a given idempotency key.
func (s *RedisStore) makeLockKey(key string) string {
	return s.makeKey(key) + ":lock"
}

// Get retrieves a cached response by key from Redis.
// Returns the cached record, true if found, and any error.
// If not found, returns false and nil error.
func (s *RedisStore) Get(ctx context.Context, key string) (config.IdempotencyRecord, bool, error) {
	data, err := s.client.Get(ctx, s.makeKey(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return config.IdempotencyRecord{}, false, nil
	}
	if err != nil {
		return config.IdempotencyRecord{}, false, err
	}

	var record redisRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return config.IdempotencyRecord{}, false, fmt.Errorf("failed to unmarshal idempotency record: %w", err)
	}

	return config.IdempotencyRecord{
		StatusCode: record.StatusCode,
		Headers:    record.Headers,
		Body:       record.Body,
		CreatedAt:  record.CreatedAt,
	}, true, nil
}

// Set stores a response in Redis with the given TTL.
// Returns an error if the operation fails.
func (s *RedisStore) Set(ctx context.Context, key string, record config.IdempotencyRecord, ttl time.Duration) error {
	redisRecord := redisRecord{
		StatusCode: record.StatusCode,
		Headers:    record.Headers,
		Body:       record.Body,
		CreatedAt:  record.CreatedAt,
	}

	data, err := json.Marshal(redisRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal idempotency record: %w", err)
	}

	return s.client.Set(ctx, s.makeKey(key), data, ttl).Err()
}

// Lock acquires an exclusive lock for the given key using Redis SET NX (set if not exists).
// Returns true if the lock was acquired, false if the key is already locked.
// The lock automatically expires after the configured lockTTL to prevent deadlocks.
func (s *RedisStore) Lock(ctx context.Context, key string) (bool, error) {
	lockKey := s.makeLockKey(key)
	// Use SET NX to atomically set only if not exists
	ok, err := s.client.SetNX(ctx, lockKey, "1", s.lockTTL).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// Unlock releases the lock for the given key.
// Returns an error if the unlock operation fails.
func (s *RedisStore) Unlock(ctx context.Context, key string) error {
	lockKey := s.makeLockKey(key)
	return s.client.Del(ctx, lockKey).Err()
}
