// Package zerohttpcontrib provides optional adapters and extensions for zerohttp.
//
// This module contains integrations that extend zerohttp's core functionality
// while maintaining its zero-dependency philosophy. Each subdirectory is a
// standalone module that can be imported independently.
//
// # Structure
//
//	adapters/       # Interface adapters (logging, OpenAPI, etc.)
//	extensions/     # Protocol extensions (HTTP/3, WebTransport, etc.)
//	middleware/     # Pluggable middleware adapters
//
// # Adapters
//
//   - adapters/huma - OpenAPI 3.1 via Huma
//   - adapters/zerolog - Structured logging via zerolog
//
// # Extensions
//
//   - extensions/autocert - Automatic TLS via Let's Encrypt
//   - extensions/http3 - HTTP/3 support via quic-go
//   - extensions/webtransport - WebTransport support
//   - extensions/websocket - WebSocket support via gorilla/websocket
//
// # Middleware
//
//   - middleware/cache - Redis store for response caching
//   - middleware/compress - Brotli/Zstd compression
//   - middleware/idempotency - Redis store for idempotency
//   - middleware/jwtauth - JWT support via lestrrat-go/jwx
//   - middleware/ratelimit - Redis store for distributed rate limiting
//   - middleware/tracer - OpenTelemetry tracer adapter
//
// # Installation
//
// Import only the modules you need:
//
//	go get github.com/alexferl/zerohttp-contrib/adapters/zerolog
//	go get github.com/alexferl/zerohttp-contrib/middleware/tracer
//
// The modules in this repository demonstrate how to integrate specific
// third-party libraries with zerohttp. You are encouraged to use these as
// reference implementations and create your own adapters for your specific needs.
//
// See https://github.com/alexferl/zerohttp for the core library.
package zerohttpcontrib
