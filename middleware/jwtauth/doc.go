// Package jwtauth provides JWT authentication middleware for zerohttp.
//
// This middleware handles JWT token validation and extraction,
// supporting various JWT signing algorithms and key management strategies.
// It integrates with storage.Storage for token revocation.
//
// Features:
//   - JWT token validation (RS256, HS256, ES256, EdDSA, etc.)
//   - JWK/JWKS key fetching and caching
//   - Token revocation support using storage.Storage interface
//   - Configurable token extraction from headers, cookies, or query params
//
// Storage Pattern:
//
// This package follows the same storage pattern as cache and idempotency middlewares.
// You provide a storage.Storage implementation (from github.com/alexferl/zerohttp/storage),
// and it's wrapped internally by StorageAdapter for revocation operations.
//
// Available storage implementations:
//   - storage.RedisStorage - Redis-backed storage (production)
//   - storage.MemoryStorage - In-memory storage (testing only)
//   - Custom implementations of storage.Storage
//
// Example usage with Redis:
//
//	import (
//	    "github.com/alexferl/zerohttp-contrib/middleware/jwtauth"
//	    zstorage "github.com/alexferl/zerohttp-contrib/storage"
//	    "github.com/redis/go-redis/v9"
//	    "github.com/lestrrat-go/jwx/v3/jwk"
//	)
//
//	// Create key set
//	rawKey := []byte("your-secret-key-at-least-32-bytes-long!")
//	key, _ := jwk.Import(rawKey)
//	keySet := jwk.NewSet()
//	keySet.AddKey(key)
//
//	// Create Redis storage
//	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	storage := zstorage.NewRedisStorage(redisClient, zstorage.RedisStorageConfig{
//	    KeyPrefix: "myapp:",
//	})
//
//	// Create token store
//	cfg := jwtauth.Config{
//	    KeySet:  keySet,
//	    Storage: storage,
//	}
//	tokenStore := jwtauth.NewTokenStore(cfg)
//
//	// Use with zerohttp
//	jwtCfg := zjwtauth.Config{
//	    Store: tokenStore,
//	}
//	app.Use(zjwtauth.New(jwtCfg))
//
// Example usage with in-memory storage (testing only):
//
//	storage := storage.NewMemoryStorage()
//	cfg := jwtauth.Config{
//	    KeySet:  keySet,
//	    Storage: storage,
//	}
//	tokenStore := jwtauth.NewTokenStore(cfg)
//
// Installation:
//
//	go get github.com/alexferl/zerohttp-contrib/middleware/jwtauth
//
// This package uses lestrrat-go/jwx for JWT processing.
// See https://github.com/lestrrat-go/jwx for more information.
package jwtauth
