# zerolog Adapter Example

This example demonstrates how to use the [zerolog](https://github.com/rs/zerolog) adapter with zerohttp for high-performance structured logging.

## Features

- Zero-allocation JSON logging
- Console output with pretty formatting for development
- Full implementation of zerohttp's `log.Logger` interface

## Running the Example

```bash
# Install dependencies
go mod tidy

# Run the server
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint     | Description                          |
|--------------|--------------------------------------|
| `GET /`      | Returns a JSON response, logs info   |
| `GET /error` | Returns an error, logs error message |
