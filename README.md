# Webhook Proxy

A proxy service for receiving webhooks and forwarding them to multiple destinations.

## Features

- Reception of webhooks on configurable endpoints
- Forwarding of webhooks to multiple destinations
- Detailed logging of requests and responses
- Configuration via YAML file or environment variables
- Configuration validation
- Retry mechanism for failed destinations
- Metrics to monitor performance
- Health and metrics endpoints

## Installation

### Precompiled Binaries

You can download precompiled binaries for your operating system from the [releases page](https://github.com/flemzord/webhook-proxy/releases).

### Building from Source

```bash
# Clone the repository
git clone https://github.com/flemzord/webhook-proxy.git
cd webhook-proxy

# Build the project
go build -o webhook-proxy ./cmd/webhook-proxy

# Run the service
./webhook-proxy -config config.yaml
```

### Using Docker

```bash
# Download the image
docker pull ghcr.io/flemzord/webhook-proxy:latest

# Run the container with your configuration file
docker run -v $(pwd)/config.yaml:/app/config/config.yaml -p 8080:8080 ghcr.io/flemzord/webhook-proxy:latest
```

## Configuration

### YAML File

Create a YAML configuration file based on the provided example (`config.example.yaml`):

```yaml
# Server configuration
server:
  host: "0.0.0.0"
  port: 8080

# Logging configuration
logging:
  level: "info"
  format: "json"
  output: "stdout"
  file_path: ""

# Endpoints configuration
endpoints:
  - path: "/webhook/github"
    destinations:
      - url: "https://example.com/github-webhook"
        headers:
          X-Custom-Header: "custom-value"
      - url: "https://backup-service.example.com/github-events"
```

### Environment Variables

You can also configure the service using environment variables, which take precedence over the values in the YAML file:

| Variable | Description | Example |
|----------|-------------|---------|
| `WEBHOOK_PROXY_SERVER_HOST` | Server host | `0.0.0.0` |
| `WEBHOOK_PROXY_SERVER_PORT` | Server port | `8080` |
| `WEBHOOK_PROXY_LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |
| `WEBHOOK_PROXY_LOG_FORMAT` | Logging format (json, text) | `json` |
| `WEBHOOK_PROXY_LOG_OUTPUT` | Logging destination (stdout, stderr, file) | `stdout` |
| `WEBHOOK_PROXY_LOG_FILE_PATH` | Logging file path (required if output=file) | `/var/log/webhook-proxy.log` |

**Note**: Endpoints must be configured via the YAML file.

## Usage

1. Start the service with your configuration file:
   ```bash
   ./webhook-proxy -config config.yaml
   ```

2. Send webhooks to the configured endpoints, for example:
   ```bash
   curl -X POST http://localhost:8080/webhook/github -d '{"event":"push","repository":"example"}'
   ```

3. The service will forward the request to all destinations configured for that endpoint.

## System Endpoints

In addition to the configured webhook endpoints, the service exposes the following system endpoints:

### Metrics

- **GET /metrics**: Returns service metrics in JSON format, including:
  - Total number of requests
  - Number of successful requests
  - Number of failed requests
  - Number of retries
  - Success rate
  - Metrics per destination

- **POST /metrics/reset**: Resets all metrics

### Health

- **GET /health**: Returns the health status of the service

Example response from the `/metrics` endpoint:
```json
{
  "global": {
    "total_requests": 42,
    "successful_requests": 40,
    "failed_requests": 2,
    "retries": 1,
    "success_rate": 95.23
  },
  "endpoints": {
    "/webhook/github": {
      "total_requests": 42,
      "successful_requests": 40,
      "failed_requests": 2,
      "retries": 1,
      "avg_response_time_ms": 125.5,
      "status_codes": {
        "200": 40,
        "500": 2
      },
      "destinations": {
        "https://example.com/github-webhook": {
          "total_requests": 21,
          "successful_requests": 20,
          "failed_requests": 1,
          "retries": 0,
          "avg_response_time_ms": 100.2
        },
        "https://backup-service.example.com/github-events": {
          "total_requests": 21,
          "successful_requests": 20,
          "failed_requests": 1,
          "retries": 1,
          "avg_response_time_ms": 150.8,
          "last_error": "connection timeout",
          "last_error_time": "2023-01-01T12:00:00Z"
        }
      }
    }
  },
  "timestamp": "2023-01-01T12:30:00Z"
}
```

## Development

### Prerequisites

- Go 1.21 or higher
- Make
- golangci-lint (for linting)
- goreleaser (for creating releases)

You can install the development dependencies with:

```bash
make dev-deps
```

### Make Commands

- `make build`: Compiles the application
- `make test`: Runs the tests
- `make lint`: Checks the code with golangci-lint
- `make lint-fix`: Automatically fixes linting issues
- `make release-snapshot`: Creates a snapshot release with GoReleaser (for testing)
- `make release`: Creates an official release with GoReleaser

### Creating a Release

To create a new release:

1. Make sure all tests pass and linting is correct
   ```bash
   make lint test
   ```

2. Create a Git tag for the new version
   ```bash
   git tag -a v1.0.0 -m "Version 1.0.0"
   git push origin v1.0.0
   ```

3. Use GoReleaser to create the release
   ```bash
   make release
   ```

This command will:
- Create binaries for different platforms (Linux, macOS, Windows)
- Generate archives containing the binaries and documentation
- Create and push a Docker image
- Generate a changelog based on commits

For more details on development and upcoming features, see the [TODO.md](TODO.md) file.

## License

This project is licensed under the MIT License. See the LICENSE file for details.
