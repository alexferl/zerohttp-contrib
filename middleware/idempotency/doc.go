// Package idempotency provides idempotency middleware for zerohttp.
//
// This middleware ensures that requests are only processed once,
// preventing duplicate operations when clients retry failed requests.
// It uses Redis to track processed requests and their responses.
//
// Features:
//   - Redis-backed idempotency key storage
//   - Configurable TTL for idempotency keys
//   - Automatic cleanup of expired keys
//
// Installation:
//
//	go get github.com/alexferl/zerohttp-contrib/middleware/idempotency
//
// This middleware requires a Redis connection. See the storage package
// for Redis configuration options.
package idempotency
