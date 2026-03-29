// Package storage provides storage adapters that implement the zerohttp/storage.Storage interface.
package storage

import (
	"context"
	"errors"
	"time"

	"github.com/alexferl/zerohttp/storage"
	"github.com/redis/go-redis/v9"
)

// Client is the minimal interface for Redis operations used by RedisStorage.
// This interface accepts:
//   - *redis.Client (single-node Redis)
//   - *redis.ClusterClient (Redis Cluster)
//   - redis.UniversalClient (abstract client for any Redis deployment)
//   - Custom mocks for testing
type Client interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
	SetArgs(ctx context.Context, key string, value any, a redis.SetArgs) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
	Close() error
}

// RedisStorage implements storage.Storage using Redis as the backend.
// It provides a generic key-value storage layer that can be used as a
// building block for custom Store implementations.
//
// RedisStorage also implements:
//   - storage.Locker (Lock/Unlock for distributed locking)
//   - storage.Inspector (TTL for key introspection)
//
// The client can be *redis.Client, *redis.ClusterClient, redis.UniversalClient,
// or any type implementing the Client interface.
type RedisStorage struct {
	client    Client
	keyPrefix string
	lockTTL   time.Duration
}

// RedisStorageConfig configures a RedisStorage.
type RedisStorageConfig struct {
	// KeyPrefix is prepended to all keys. Optional.
	KeyPrefix string
	// LockTTL is the default TTL for distributed locks. Defaults to 30 seconds.
	LockTTL time.Duration
}

// NewRedisStorage creates a new Redis-backed storage adapter.
// The client can be *redis.Client, *redis.ClusterClient, redis.UniversalClient,
// or any type implementing the Client interface.
// Config is passed as variadic struct (allowing inline construction).
func NewRedisStorage(client Client, cfg ...RedisStorageConfig) *RedisStorage {
	s := &RedisStorage{
		client:  client,
		lockTTL: 30 * time.Second, // Default lock TTL
	}

	if len(cfg) > 0 {
		c := cfg[0]
		s.keyPrefix = c.KeyPrefix
		if c.LockTTL > 0 {
			s.lockTTL = c.LockTTL
		}
	}

	return s
}

// makeKey creates a Redis key with optional prefix.
func (s *RedisStorage) makeKey(key string) string {
	if s.keyPrefix != "" {
		return s.keyPrefix + key
	}
	return key
}

// makeLockKey creates the lock key for a given storage key.
func (s *RedisStorage) makeLockKey(key string) string {
	return s.makeKey(key) + ":lock"
}

// Get retrieves a value by key from Redis.
// Returns the value, true if found, and any error.
// If not found, returns nil, false, nil error.
func (s *RedisStorage) Get(ctx context.Context, key string) ([]byte, bool, error) {
	data, err := s.client.Get(ctx, s.makeKey(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

// Set stores a value with the given TTL.
// Returns an error if the operation fails.
func (s *RedisStorage) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return s.client.Set(ctx, s.makeKey(key), val, ttl).Err()
}

// Delete removes a key from Redis.
// Returns an error if the operation fails.
func (s *RedisStorage) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, s.makeKey(key)).Err()
}

// Lock acquires an exclusive lock for the given key using Redis SET NX.
// Returns true if the lock was acquired, false if already locked.
// The lock automatically expires after the configured lockTTL to prevent deadlocks.
// This implements the storage.Locker interface.
func (s *RedisStorage) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	lockKey := s.makeLockKey(key)
	cmd := s.client.SetArgs(ctx, lockKey, "1", redis.SetArgs{
		Mode: "NX",
		TTL:  ttl,
	})
	_, err := cmd.Result()
	if errors.Is(err, redis.Nil) {
		// Key already exists - lock not acquired
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Lock acquired
	return true, nil
}

// Unlock releases the lock for the given key.
// Returns an error if the unlock operation fails.
// This implements the storage.Locker interface.
func (s *RedisStorage) Unlock(ctx context.Context, key string) error {
	lockKey := s.makeLockKey(key)
	return s.client.Del(ctx, lockKey).Err()
}

// TTL returns the remaining time-to-live for a key.
// Returns 0 if the key does not exist or has no TTL.
// This implements the storage.Inspector interface.
func (s *RedisStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := s.client.TTL(ctx, s.makeKey(key)).Result()
	if err != nil {
		return 0, err
	}
	// Redis returns -1 for keys that exist but have no TTL, -2 for non-existent keys
	if ttl < 0 {
		return 0, nil
	}
	return ttl, nil
}

// Close releases resources associated with the storage.
// Returns an error if the close operation fails.
func (s *RedisStorage) Close() error {
	return s.client.Close()
}

// compile-time interface checks
var (
	_ storage.Storage   = (*RedisStorage)(nil)
	_ storage.Locker    = (*RedisStorage)(nil)
	_ storage.Inspector = (*RedisStorage)(nil)
)
