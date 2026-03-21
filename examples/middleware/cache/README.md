# Cache with Redis Example

This example demonstrates distributed HTTP response caching using Redis as the backing store. This allows cached responses to be shared across multiple server instances.

## Features

- Distributed response caching with Redis
- Automatic ETag generation (SHA256 hash of body)
- Last-Modified timestamp support
- Vary header support (cache different responses based on request headers)
- Configurable TTL per endpoint
- Cache-Control header management

## Prerequisites

### Start Redis with Docker

```bash
# Start Redis container
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Verify Redis is running
docker ps

# View Redis logs
docker logs redis
```

### Or use Docker Compose

```yaml
version: '3.8'
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

```bash
docker-compose up -d
```

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint | Cache TTL | Description |
|----------|-----------|-------------|
| `GET /api/public/data` | 30s | Public API with ETag support |
| `GET /api/users/{id}` | 60s | Private user data with Vary headers |
| `GET /api/live` | None | Health check (not cached) |
| `GET /api/config` | 1h | Static config (immutable) |
| `GET /page/info` | 2m | HTML page caching |
| `GET /api/stats` | 10s | Short-term cached stats |

## Test Commands

### Test cache hit/miss

```bash
# First request - cache miss
curl -i http://localhost:8080/api/public/data

# Check for X-Cache header (if enabled):
# X-Cache: MISS

# Second request - cache hit
curl -i http://localhost:8080/api/public/data

# Response served from cache (timestamp unchanged)
```

### Test ETag support

```bash
# Get response with ETag
ETAG=$(curl -s -i http://localhost:8080/api/public/data | grep -i etag | awk '{print $2}' | tr -d '\r')

# Conditional request with If-None-Match
curl -i -H "If-None-Match: $ETAG" http://localhost:8080/api/public/data

# Should return 304 Not Modified
```

### Test Vary header (different caches per Accept)

```bash
# Request JSON
curl -i -H "Accept: application/json" http://localhost:8080/api/users/123

# Request XML (different cached response)
curl -i -H "Accept: application/xml" http://localhost:8080/api/users/123
```

### Verify caching is working

```bash
# Make multiple requests and check the timestamp
for i in {1..5}; do
  curl -s http://localhost:8080/api/public/data | jq '.timestamp'
  sleep 1
done

# All timestamps should be identical (within the 30s cache window)
```

### View cached keys in Redis

```bash
# Connect to Redis
docker exec -it redis redis-cli

# List cache keys
KEYS zerohttp:cache:*

# View a specific key
GET zerohttp:cache:<key>

# Check TTL
TTL zerohttp:cache:<key>
```

## Cleanup

```bash
# Stop and remove Redis container
docker stop redis
docker rm redis
```
