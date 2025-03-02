FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o webhook-proxy

# Create a minimal image
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/webhook-proxy /app/webhook-proxy

# Create a directory for configuration
RUN mkdir -p /app/config

# Copy the default configuration
COPY config.yaml /app/config/config.yaml

# Expose the port
EXPOSE 8080

# Set the entrypoint
ENTRYPOINT ["/app/webhook-proxy", "--config", "/app/config/config.yaml"] 