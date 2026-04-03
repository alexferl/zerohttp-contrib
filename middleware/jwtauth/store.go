package jwtauth

import (
	"context"
	"fmt"
	"time"

	"github.com/alexferl/zerohttp/storage"
)

// StorageAdapter wraps storage.Storage to provide token revocation functionality.
// This adapter allows jwtauth to use any storage backend that implements storage.Storage
// (e.g., Redis, PostgreSQL, in-memory) for token revocation.
//
// The adapter handles key namespacing with prefixes for tokens and sessions.
//
// Example usage:
//
//	// Create Redis storage
//	redisStorage := storage.NewRedisStorage(redisClient, storage.RedisStorageConfig{
//	    KeyPrefix: "app:",
//	})
//
//	// Create config with storage
//	cfg := jwtauth.Config{
//	    KeySet:  keySet,
//	    Storage: redisStorage,
//	}
//	store := jwtauth.NewTokenStore(cfg)
type StorageAdapter struct {
	storage storage.Storage
	prefix  string
}

// StorageAdapterConfig configures the StorageAdapter.
type StorageAdapterConfig struct {
	// KeyPrefix is the prefix for keys.
	// Default: "jwt"
	KeyPrefix string
}

// DefaultStorageAdapterConfig is the default configuration for StorageAdapter.
var DefaultStorageAdapterConfig = StorageAdapterConfig{
	KeyPrefix: "jwt",
}

// NewStorageAdapter creates a StorageAdapter from a storage.Storage backend.
//
// Configuration is applied via variadic StorageAdapterConfig (allowing inline construction).
// If no config is provided, defaults are used.
// If multiple configs are provided, the first one is used.
func NewStorageAdapter(s storage.Storage, cfg ...StorageAdapterConfig) *StorageAdapter {
	c := DefaultStorageAdapterConfig
	if len(cfg) > 0 {
		userCfg := cfg[0]
		if userCfg.KeyPrefix != "" {
			c.KeyPrefix = userCfg.KeyPrefix
		}
	}
	return &StorageAdapter{
		storage: s,
		prefix:  c.KeyPrefix,
	}
}

func (a *StorageAdapter) tokenKey(key string) string {
	return a.prefix + ":token:" + key
}

func (a *StorageAdapter) sessionKey(sid string) string {
	return a.prefix + ":session:" + sid
}

// RevokeToken marks a specific token as revoked.
func (a *StorageAdapter) RevokeToken(ctx context.Context, key string, ttl time.Duration) error {
	return a.storage.Set(ctx, a.tokenKey(key), []byte("1"), ttl)
}

// RevokeSession marks an entire session as revoked.
func (a *StorageAdapter) RevokeSession(ctx context.Context, sid string, ttl time.Duration) error {
	return a.storage.Set(ctx, a.sessionKey(sid), []byte("1"), ttl)
}

// IsTokenRevoked checks if a specific token has been revoked.
func (a *StorageAdapter) IsTokenRevoked(ctx context.Context, key string) (bool, error) {
	_, found, err := a.storage.Get(ctx, a.tokenKey(key))
	if err != nil {
		return false, fmt.Errorf("failed to check token revocation: %w", err)
	}
	return found, nil
}

// IsSessionRevoked checks if a session has been revoked.
func (a *StorageAdapter) IsSessionRevoked(ctx context.Context, sid string) (bool, error) {
	_, found, err := a.storage.Get(ctx, a.sessionKey(sid))
	if err != nil {
		return false, fmt.Errorf("failed to check session revocation: %w", err)
	}
	return found, nil
}

// Close closes the underlying storage.
func (a *StorageAdapter) Close() error {
	return a.storage.Close()
}
