# JWT Authentication with lestrrat-go/jwx Example

This example demonstrates JWT authentication using the `github.com/lestrrat-go/jwx` library with Redis-backed token revocation.

## Features

- JWT authentication using lestrrat-go/jwx v3
- Access and refresh token support
- Token rotation with single-use refresh tokens
- Redis-based token revocation (logout)
- Session-based revocation (refresh revokes old session)
- Protected and public endpoints

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

| Endpoint             | Auth Required | Description                 |
|----------------------|---------------|-----------------------------|
| `POST /login`        | No            | Authenticate and get tokens |
| `POST /auth/refresh` | No            | Refresh access token        |
| `POST /auth/logout`  | No            | Revoke refresh token        |
| `GET /api/profile`   | Yes           | Get user profile            |

## Credentials

- Username: `alice`
- Password: `secret`

## Test Commands

### Login and extract tokens with jq

```bash
TOKENS=$(curl -s -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"secret"}')
ACCESS_TOKEN=$(echo $TOKENS | jq -r '.access_token')
REFRESH_TOKEN=$(echo $TOKENS | jq -r '.refresh_token')
echo "Access token: $ACCESS_TOKEN"
echo "Refresh token: $REFRESH_TOKEN"
```

### Access protected endpoint

```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/profile
```

### Refresh tokens (revokes old session)

```bash
NEW_TOKENS=$(curl -s -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
NEW_ACCESS_TOKEN=$(echo $NEW_TOKENS | jq -r '.access_token')
NEW_REFRESH_TOKEN=$(echo $NEW_TOKENS | jq -r '.refresh_token')
echo "New access token: $NEW_ACCESS_TOKEN"
echo "New refresh token: $NEW_REFRESH_TOKEN"
```

### Try old token after refresh (fails - session revoked)

```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/profile
```

Expected response:
```json
{"error":"Invalid Token","detail":"The provided token is invalid or has expired"}
```

### Logout (revoke refresh token)

```bash
curl -X POST http://localhost:8080/auth/logout \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$NEW_REFRESH_TOKEN\"}"
```

### Try revoked refresh token (fails)

```bash
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$NEW_REFRESH_TOKEN\"}"
```

Expected response:
```json
{"error":"Token Revoked","detail":"token has been revoked"}
```

### Access without token (fails)

```bash
curl http://localhost:8080/api/profile
```

Expected response:
```json
{"error":"Missing Authorization Token","detail":"Request is missing the Authorization header with Bearer token"}
```

### View revoked tokens in Redis

```bash
# Connect to Redis
docker exec -it redis redis-cli

# List revoked tokens
KEYS jwt:*

# View a specific key
GET jwt:token:<key>

# Check TTL
TTL jwt:token:<key>

# List revoked sessions
KEYS jwt:session:*
```

## Cleanup

```bash
# Stop and remove Redis container
docker stop redis
docker rm redis
```