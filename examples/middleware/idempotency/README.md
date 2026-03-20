# Idempotency Redis Store Example

This example demonstrates how to use the Redis-backed idempotency store for distributed idempotency across multiple server instances.

## Features

- Distributed idempotency using Redis
- Automatic lock management for in-flight requests
- Configurable TTL for cached responses

## Prerequisites

- Redis server running on `localhost:6379`

## Running the Example

```bash
# Start Redis
docker run -d -p 6379:6379 redis:latest

# Install dependencies
go mod tidy

# Run the server
go run .
```

The server starts on `http://localhost:8080`.

## Testing Idempotency

Make the same request twice with the same `Idempotency-Key`:

```bash
# First request - processes the payment
curl -X POST http://localhost:8080/payments \
  -H "Idempotency-Key: payment-123" \
  -H "Content-Type: application/json" \
  -d '{"amount": 100}'

# Second request - returns cached response
curl -X POST http://localhost:8080/payments \
  -H "Idempotency-Key: payment-123" \
  -H "Content-Type: application/json" \
  -d '{"amount": 100}'
```