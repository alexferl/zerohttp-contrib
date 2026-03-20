# zerohttp-contrib Examples

This directory contains example applications demonstrating how to use the adapters and extensions in zerohttp-contrib.

Each example has its own `go.mod` file since they depend on third-party libraries.

## Directory Structure

### `adapters/` - Interface Adapters

Examples showing how to integrate with external libraries via adapters.

- [**`huma/`**](adapters/huma/) - OpenAPI 3.1 API with Huma
- [**`zerolog/`**](adapters/zerolog/) - Structured logging with zerolog

### `extensions/` - Protocol Extensions

Examples demonstrating protocol extensions that require third-party libraries.

- [**`autocert/`**](extensions/autocert/) - Automatic TLS via Let's Encrypt
- [**`http3/`**](extensions/http3/) - HTTP/3 and QUIC support
- [**`http3_autocert/`**](extensions/http3_autocert/) - HTTP/3 with AutoTLS
- [**`websocket/`**](extensions/websocket/) - WebSocket support
- [**`webtransport/`**](extensions/webtransport/) - WebTransport protocol
- [**`webtransport_autocert/`**](extensions/webtransport_autocert/) - WebTransport with AutoTLS

### `middleware/` - Middleware with External Dependencies

Examples of middleware that integrate with external libraries.

- [**`cache/`**](middleware/cache/) - Redis-backed HTTP caching
- [**`compress/`**](middleware/compress/) - Brotli and Zstd compression
- [**`idempotency/`**](middleware/idempotency/) - Redis-backed idempotency
- [**`jwtauth/`**](middleware/jwtauth/) - JWT authentication with lestrrat-go/jwx
- [**`ratelimit/`**](middleware/ratelimit/) - Redis-backed rate limiting
- [**`tracer/`**](middleware/tracer/) - OpenTelemetry tracing

## Running Examples

All examples in this directory have their own `go.mod` files and require external dependencies.

```bash
cd <example-directory>
go mod tidy
go run .
```

For example:

```bash
cd adapters/zerolog
go mod tidy
go run .
```

## Common Patterns

### Using an Adapter

```go
import (
    zh "github.com/alexferl/zerohttp"
    zclog "github.com/alexferl/zerohttp-contrib/adapters/zerolog"
)

logger := zclog.NewConsole()
app := zh.New(config.Config{Logger: logger})
```

### Using an Extension

```go
import (
    zh "github.com/alexferl/zerohttp"
    zcws "github.com/alexferl/zerohttp-contrib/extensions/websocket"
)

server := zcws.NewServer(app)
server.Start()
```

### Using Middleware with External Store

```go
import (
    "github.com/alexferl/zerohttp/middleware"
    "github.com/alexferl/zerohttp-contrib/middleware/ratelimit"
    "github.com/redis/go-redis/v9"
)

client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
store := ratelimit.NewRedisStore(client, config.TokenBucket, time.Minute, 100)

app.Use(middleware.RateLimit(config.RateLimitConfig{Store: store}))
```

See individual example directories for complete, runnable code.
