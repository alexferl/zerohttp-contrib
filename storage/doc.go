// Package storage provides storage adapters for zerohttp.
//
// This package implements storage interfaces defined by zerohttp,
// providing concrete storage backends that can be used by various
// middleware components.
//
// Storage Backends:
//
//   - Redis - High-performance key-value store using go-redis
//
// Features:
//   - Implements zerohttp/storage.Storage interface
//   - Connection pooling and retry logic
//   - Configurable timeouts and TTL
//
// Installation:
//
//	go get github.com/alexferl/zerohttp-contrib/storage
//
// This package is typically used as a dependency by other middleware
// rather than being used directly. See cache, idempotency, ratelimit,
// and jwtauth packages for usage examples.
package storage
