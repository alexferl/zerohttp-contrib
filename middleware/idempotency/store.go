package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alexferl/zerohttp/middleware/idempotency"
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
	SetArgs(ctx context.Context, key string, value any, a redis.SetArgs) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Close() error
}

// RedisStoreConfig configures the RedisStore.
type RedisStoreConfig struct {
	// KeyPrefix is the prefix for idempotency keys.
	// Default: ""
	KeyPrefix string

	// LockTTL is the TTL for distributed locks.
	// Default: 30s
	LockTTL time.Duration
}

// DefaultRedisStoreConfig is the default configuration for RedisStore.
var DefaultRedisStoreConfig = RedisStoreConfig{
	KeyPrefix: "",
	LockTTL:   30 * time.Second,
}

// RedisStore implements idempotency.Store using Redis for distributed
// idempotency across multiple server instances.
type RedisStore struct {
	client    RedisClient
	keyPrefix string
	lockTTL   time.Duration
}

// redisRecord is a JSON-serializable representation of idempotency.Record.
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
//
// Configuration is applied via variadic RedisStoreConfig (allowing inline construction).
// If no config is provided, defaults are used.
// If multiple configs are provided, the first one is used.
func NewRedisStore(client RedisClient, cfg ...RedisStoreConfig) *RedisStore {
	c := DefaultRedisStoreConfig
	if len(cfg) > 0 {
		userCfg := cfg[0]
		if userCfg.KeyPrefix != "" {
			c.KeyPrefix = userCfg.KeyPrefix
		}
		if userCfg.LockTTL != 0 {
			c.LockTTL = userCfg.LockTTL
		}
	}

	return &RedisStore{
		client:    client,
		keyPrefix: c.KeyPrefix,
		lockTTL:   c.LockTTL,
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
func (s *RedisStore) Get(ctx context.Context, key string) (idempotency.Record, bool, error) {
	data, err := s.client.Get(ctx, s.makeKey(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return idempotency.Record{}, false, nil
	}
	if err != nil {
		return idempotency.Record{}, false, err
	}

	var record redisRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return idempotency.Record{}, false, fmt.Errorf("failed to unmarshal idempotency record: %w", err)
	}

	return idempotency.Record{
		StatusCode: record.StatusCode,
		Headers:    record.Headers,
		Body:       record.Body,
		CreatedAt:  record.CreatedAt,
	}, true, nil
}

// Set stores a response in Redis with the given TTL.
// Returns an error if the operation fails.
func (s *RedisStore) Set(ctx context.Context, key string, record idempotency.Record, ttl time.Duration) error {
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
	cmd := s.client.SetArgs(ctx, lockKey, "1", redis.SetArgs{
		Mode: "NX",
		TTL:  s.lockTTL,
	})
	_, err := cmd.Result()
	if errors.Is(err, redis.Nil) {
		// Key already exists - lock not acquired
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Unlock releases the lock for the given key.
// Returns an error if the unlock operation fails.
func (s *RedisStore) Unlock(ctx context.Context, key string) error {
	lockKey := s.makeLockKey(key)
	return s.client.Del(ctx, lockKey).Err()
}

// Close closes the Redis connection.
// Returns an error if the close operation fails.
func (s *RedisStore) Close() error {
	return s.client.Close()
}
