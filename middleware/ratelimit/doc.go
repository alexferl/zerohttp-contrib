// Package ratelimit provides rate limiting middleware for zerohttp.
//
// This middleware implements distributed rate limiting using Redis,
// allowing you to control request rates across multiple server instances.
//
// Features:
//   - Redis-backed distributed rate limiting
//   - Configurable rate limit windows and burst sizes
//   - Per-client rate limiting based on IP or custom keys
//   - Sliding window algorithm for accurate limiting
//
// Installation:
//
//	go get github.com/alexferl/zerohttp-contrib/middleware/ratelimit
//
// This middleware requires a Redis connection. See the storage package
// for Redis configuration options.
package ratelimit
