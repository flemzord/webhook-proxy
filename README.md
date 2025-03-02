# Webhook Proxy

A service that receives webhooks and forwards them to multiple configured destinations, with detailed logging of each step.

## Features

- Reception of webhooks on configurable endpoints
- Forwarding of webhooks to multiple destinations
- Configuration via YAML file
- Detailed logging of received and forwarded webhooks
- Error handling and retry mechanism

## Installation

```bash
# Clone the repository
git clone https://github.com/flemzord/webhook-proxy.git
cd webhook-proxy

# Build the application
go build -o webhook-proxy

# Run the application
./webhook-proxy --config config.yaml
```

## Configuration

The service is configured via a YAML file. Configuration example:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

logging:
  level: "info"
  format: "json"
  output: "stdout"

endpoints:
  - path: "/webhook/github"
    destinations:
      - url: "https://destination1.example.com/webhook"
        method: "POST"
        headers:
          Content-Type: "application/json"
        timeout: 5s
        retries: 3
```

See the [TODO.md](TODO.md) file for more details on the development plan and planned features.

## License

MIT
