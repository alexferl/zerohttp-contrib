# zerohttp-contrib [![Go Reference](https://pkg.go.dev/badge/github.com/alexferl/zerohttp-contrib.svg)](https://pkg.go.dev/github.com/alexferl/zerohttp-contrib) [![Go Report Card](https://goreportcard.com/badge/github.com/alexferl/zerohttp-contrib)](https://goreportcard.com/report/github.com/alexferl/zerohttp-contrib) [![Coverage Status](https://coveralls.io/repos/github/alexferl/zerohttp-contrib/badge.svg?branch=master)](https://coveralls.io/github/alexferl/zerohttp-contrib?branch=master)

Optional adapters and extensions for [zerohttp](https://github.com/alexferl/zerohttp). Each module is standalone - import only what you need.

## Installation

```bash
go get github.com/alexferl/zerohttp-contrib/<category>/<name>
```

## Adapters

| Adapter            | Description                                                     | Example                                                |
|--------------------|-----------------------------------------------------------------|--------------------------------------------------------|
| `adapters/huma`    | OpenAPI 3.1 via [Huma](https://github.com/danielgtaylor/huma)   | [examples/adapters/huma](examples/adapters/huma)       |
| `adapters/zerolog` | Structured logging via [zerolog](https://github.com/rs/zerolog) | [examples/adapters/zerolog](examples/adapters/zerolog) |

## Extensions

| Extension                 | Description                                                                                    | Example                                                              |
|---------------------------|------------------------------------------------------------------------------------------------|----------------------------------------------------------------------|
| `extensions/autocert`     | Automatic TLS via [Let's Encrypt](https://letsencrypt.org/)                                    | [examples/extensions/autocert](examples/extensions/autocert)         |
| `extensions/http3`        | HTTP/3 support via [quic-go/quic-go](https://github.com/quic-go/quic-go)                       | [examples/extensions/http3](examples/extensions/http3)               |
| `extensions/webtransport` | WebTransport support via [quic-go/webtransport-go](https://github.com/quic-go/webtransport-go) | [examples/extensions/webtransport](examples/extensions/webtransport) |
| `extensions/websocket`    | WebSocket support via [gorilla/websocket](https://github.com/gorilla/websocket)                | [examples/extensions/websocket](examples/extensions/websocket)       |

## Middleware

| Middleware               | Description                                                           | Example                                                            |
|--------------------------|-----------------------------------------------------------------------|--------------------------------------------------------------------|
| `middleware/cache`       | Redis store for response caching                                      | [examples/middleware/cache](examples/middleware/cache)             |
| `middleware/compress`    | Brotli/Zstd compression                                               | [examples/middleware/compress](examples/middleware/compress)       |
| `middleware/idempotency` | Redis store for idempotency                                           | [examples/middleware/idempotency](examples/middleware/idempotency) |
| `middleware/jwtauth`     | JWT support via [lestrrat-go/jwx](https://github.com/lestrrat-go/jwx) | [examples/middleware/jwtauth](examples/middleware/jwtauth)         |
| `middleware/ratelimit`   | Redis store for distributed limits                                    | [examples/middleware/ratelimit](examples/middleware/ratelimit)     |
| `middleware/tracer`      | OpenTelemetry tracer adapter                                          | [examples/middleware/tracer](examples/middleware/tracer)           |

## Structure

```
adapters/       # Interface adapters (logging, OpenAPI, etc.)
extensions/     # Protocol extensions (HTTP/3, WebTransport, etc.)
middleware/     # Pluggable middleware adapters
```

Each subdirectory is an independent Go module with its own `go.mod`.

## Reference Implementations

The modules in this repository demonstrate how to integrate specific third-party libraries with zerohttp.

If you need to use a different library (e.g., zap instead of zerolog, or a different Redis client), **create your own module**. Copy the adapter code from this repo, modify it for your library, and maintain it in your own project or as a separate module.

The zerohttp core is intentionally minimal with zero dependencies. This contrib repo provides examples - you are encouraged to build and maintain your own adapters for your specific needs.
