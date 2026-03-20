# Huma Adapter Example

This example demonstrates how to use the [Huma](https://github.com/danielgtaylor/huma) adapter with zerohttp to build OpenAPI-compatible REST APIs.

## Features

- OpenAPI 3.1 spec generation
- Automatic input validation
- Type-safe handlers
- Automatic documentation UI

## Running the Example

```bash
# Install dependencies
go mod tidy

# Run the server
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint               | Description           |
|------------------------|-----------------------|
| `GET /`                | Simple hello world    |
| `GET /greeting/{name}` | Personalized greeting |

## OpenAPI Documentation

Once the server is running, visit:

- OpenAPI JSON: `http://localhost:8080/openapi.json`
- Documentation UI: `http://localhost:8080/docs`
