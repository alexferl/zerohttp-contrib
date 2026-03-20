# Compression Middleware Example

This example demonstrates HTTP response compression using Brotli, Zstd, Gzip, and Deflate algorithms.

## Features

- **Brotli** (`br`): Best compression ratio, slower compression, fast decompression
- **Zstd** (`zstd`): Excellent compression with very fast decompression
- **Gzip** (`gzip`): Widely supported, good balance
- **Deflate** (`deflate`): Legacy support

The middleware automatically selects the best algorithm based on the client's `Accept-Encoding` header.

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint        | Description                                  |
|-----------------|----------------------------------------------|
| `GET /`         | HTML page with compression info              |
| `GET /api/data` | JSON API response (automatically compressed) |

## Test Commands

### Test Brotli compression

```bash
curl -H "Accept-Encoding: br" http://localhost:8080/ --output - | brotli -d
```

### Test Zstd compression

```bash
curl -H "Accept-Encoding: zstd" http://localhost:8080/ --output - | zstd -d
```

### Test Gzip compression

```bash
curl -H "Accept-Encoding: gzip" http://localhost:8080/ --output - | gzip -d
```

### Check response headers

```bash
curl -I -H "Accept-Encoding: br" http://localhost:8080/
```

Expected response:
```
Content-Encoding: br
Content-Type: text/html; charset=UTF-8
```

### Compare compression ratios

```bash
# Uncompressed
curl -s http://localhost:8080/ | wc -c

# Brotli
curl -s -H "Accept-Encoding: br" http://localhost:8080/ | wc -c

# Zstd
curl -s -H "Accept-Encoding: zstd" http://localhost:8080/ | wc -c

# Gzip
curl -s -H "Accept-Encoding: gzip" http://localhost:8080/ | wc -c
```

## Usage in Your Application

```go
import (
    "github.com/alexferl/zerohttp/config"
    "github.com/alexferl/zerohttp/middleware"
    "github.com/alexferl/zerohttp-contrib/middleware/compress"
)

app.Use(middleware.Compress(config.CompressConfig{
    Level: 6,
    Algorithms: []config.CompressionAlgorithm{
        "br",   // Brotli
        "zstd", // Zstd
        config.Gzip,
    },
    Providers: []config.CompressionProvider{
        compress.BrotliProvider{},
        compress.ZstdProvider{},
    },
}))
```
