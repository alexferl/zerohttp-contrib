package jwtauth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store defines the interface for token revocation storage.
// Users can implement this interface to plug in their preferred storage solution
// (Redis, PostgreSQL, MySQL, DynamoDB, etc.) for token revocation.
//
// The storage is used to:
//   - Track revoked tokens (by sub:jti, sub:sid, or sub:exp)
//   - Track revoked sessions (by sid)
//   - Support logout and token refresh workflows
//
// Example implementations:
//   - RedisStore: production-ready Redis-backed storage (provided by this package)
//   - SQLStorage: database-backed storage with connection pooling
//
// All methods accept context.Context for cancellation and timeout support.
type Store interface {
	// RevokeToken marks a specific token as revoked.
	// The key is typically "sub:jti", "sub:sid", or "sub:exp" - something unique to the token instance.
	// ttl indicates how long the revocation should be stored (typically token expiration time).
	//
	// Implementations should handle duplicate revocations gracefully (idempotent).
	RevokeToken(ctx context.Context, key string, ttl time.Duration) error

	// RevokeSession marks an entire session as revoked.
	// This invalidates all tokens with the matching sid claim.
	// ttl indicates how long the revocation should be stored.
	//
	// Implementations should handle duplicate revocations gracefully (idempotent).
	RevokeSession(ctx context.Context, sid string, ttl time.Duration) error

	// IsTokenRevoked checks if a specific token has been revoked.
	// Returns true if the token was revoked, false otherwise.
	IsTokenRevoked(ctx context.Context, key string) (bool, error)

	// IsSessionRevoked checks if a session has been revoked.
	// Returns true if the session was revoked, false otherwise.
	IsSessionRevoked(ctx context.Context, sid string) (bool, error)

	// Close closes the storage connection/resources.
	// Implementations should ensure this is safe to call multiple times.
	Close() error
}

// RedisStore implements the Store interface using Redis.
// This provides a production-ready distributed storage solution
// for token revocation that works across multiple server instances.
//
// The client can be *redis.Client, *redis.ClusterClient, or redis.UniversalClient.
//
// Example usage:
//
//	client := redis.NewClient(&redis.Options{
//	    Addr: "localhost:6379",
//	})
//	storage := jwtauth.NewRedisStorage(client)
//
//	cfg := jwtauth.Config{
//	    KeySet:  keySet,
//	    Storage: storage,
//	}
type RedisStore struct {
	client redis.UniversalClient
	prefix string
}

// NewRedisStore creates a new Redis-based storage.
// The prefix is used to namespace keys (default: "jwt:")
func NewRedisStore(client redis.UniversalClient, prefix string) *RedisStore {
	if prefix == "" {
		prefix = "jwt:"
	}
	return &RedisStore{
		client: client,
		prefix: prefix,
	}
}

// tokenKey generates the Redis key for a token.
func (s *RedisStore) tokenKey(key string) string {
	return fmt.Sprintf("%stoken:%s", s.prefix, key)
}

// sessionKey generates the Redis key for a session.
func (s *RedisStore) sessionKey(sid string) string {
	return fmt.Sprintf("%ssession:%s", s.prefix, sid)
}

// RevokeToken marks a specific token as revoked in Redis.
// The key is stored with the provided TTL.
func (s *RedisStore) RevokeToken(ctx context.Context, key string, ttl time.Duration) error {
	redisKey := s.tokenKey(key)
	return s.client.Set(ctx, redisKey, "1", ttl).Err()
}

// RevokeSession marks an entire session as revoked in Redis.
// The session is stored with the provided TTL.
func (s *RedisStore) RevokeSession(ctx context.Context, sid string, ttl time.Duration) error {
	redisKey := s.sessionKey(sid)
	return s.client.Set(ctx, redisKey, "1", ttl).Err()
}

// IsTokenRevoked checks if a specific token has been revoked.
// Returns true if the token exists in Redis (meaning it was revoked).
func (s *RedisStore) IsTokenRevoked(ctx context.Context, key string) (bool, error) {
	redisKey := s.tokenKey(key)
	exists, err := s.client.Exists(ctx, redisKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token revocation: %w", err)
	}
	return exists > 0, nil
}

// IsSessionRevoked checks if a session has been revoked.
// Returns true if the session exists in Redis (meaning it was revoked).
func (s *RedisStore) IsSessionRevoked(ctx context.Context, sid string) (bool, error) {
	redisKey := s.sessionKey(sid)
	exists, err := s.client.Exists(ctx, redisKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session revocation: %w", err)
	}
	return exists > 0, nil
}

// Close closes the Redis connection.
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Client returns the underlying Redis client.
// This can be used for health checks or other operations.
func (s *RedisStore) Client() redis.UniversalClient {
	return s.client
}

// Ping checks if the Redis connection is healthy.
func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}
