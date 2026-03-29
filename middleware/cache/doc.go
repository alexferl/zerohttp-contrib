// Package cache provides response caching middleware for zerohttp.
//
// This middleware caches HTTP responses using Redis, allowing you to
// reduce load on your backend services and improve response times.
//
// Features:
//   - Redis-based caching with configurable TTL
//   - Cache key generation based on request URL and headers
//   - Configurable cache control via HTTP headers
//
// Installation:
//
//	go get github.com/alexferl/zerohttp-contrib/middleware/cache
//
// This middleware requires a Redis connection. See the storage package
// for Redis configuration options.
package cache
