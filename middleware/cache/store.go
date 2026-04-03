package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alexferl/zerohttp/middleware/cache"
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
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Close() error
}

// RedisStoreConfig configures the RedisStore.
type RedisStoreConfig struct {
	// KeyPrefix is the prefix for cache keys.
	// Default: ""
	KeyPrefix string
}

// DefaultRedisStoreConfig is the default configuration for RedisStore.
var DefaultRedisStoreConfig = RedisStoreConfig{
	KeyPrefix: "",
}

// RedisStore implements cache.Store using Redis as the backend.
// This allows caching to work across multiple server instances.
type RedisStore struct {
	client RedisClient
	prefix string
}

// cacheRecord is a JSON-serializable representation of cache.Record.
type cacheRecord struct {
	StatusCode   int                 `json:"status_code"`
	Headers      map[string][]string `json:"headers"`
	Body         []byte              `json:"body"`
	ETag         string              `json:"etag"`
	LastModified time.Time           `json:"last_modified"`
	VaryHeaders  map[string]string   `json:"vary_headers"`
}

// NewRedisStore creates a new Redis-backed cache store.
// This allows caching to work across multiple server instances.
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
	}

	return &RedisStore{
		client: client,
		prefix: c.KeyPrefix,
	}
}

// makeKey creates a Redis key with optional prefix.
func (s *RedisStore) makeKey(key string) string {
	if s.prefix != "" {
		return s.prefix + ":" + key
	}
	return key
}

// Get retrieves a cached response by key from Redis.
// Returns the cached record, true if found, and any error.
// If not found, returns false and nil error.
func (s *RedisStore) Get(ctx context.Context, key string) (cache.Record, bool, error) {
	data, err := s.client.Get(ctx, s.makeKey(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return cache.Record{}, false, nil
	}
	if err != nil {
		return cache.Record{}, false, err
	}

	var record cacheRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return cache.Record{}, false, fmt.Errorf("failed to unmarshal cache record: %w", err)
	}

	return cache.Record{
		StatusCode:   record.StatusCode,
		Headers:      record.Headers,
		Body:         record.Body,
		ETag:         record.ETag,
		LastModified: record.LastModified,
		VaryHeaders:  record.VaryHeaders,
	}, true, nil
}

// Set stores a response in Redis with the given TTL.
// Returns an error if the operation fails.
func (s *RedisStore) Set(ctx context.Context, key string, record cache.Record, ttl time.Duration) error {
	redisRecord := cacheRecord{
		StatusCode:   record.StatusCode,
		Headers:      record.Headers,
		Body:         record.Body,
		ETag:         record.ETag,
		LastModified: record.LastModified,
		VaryHeaders:  record.VaryHeaders,
	}

	data, err := json.Marshal(redisRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal cache record: %w", err)
	}

	return s.client.Set(ctx, s.makeKey(key), data, ttl).Err()
}

// Delete removes a cached response by key from Redis.
// Returns an error if the operation fails.
func (s *RedisStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, s.makeKey(key)).Err()
}

// Close closes the Redis connection.
// Returns an error if the close operation fails.
func (s *RedisStore) Close() error {
	return s.client.Close()
}
