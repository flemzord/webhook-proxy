.PHONY: all build clean test lint lint-fix help release release-snapshot

# Default target
all: lint test build

# Build the application
build:
	@echo "Building webhook-proxy..."
	@go build -o webhook-proxy ./cmd/webhook-proxy

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f webhook-proxy
	@rm -rf dist/

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# Fix linting issues automatically where possible
lint-fix:
	@echo "Fixing linting issues..."
	@golangci-lint run --fix ./...

# Install development dependencies
dev-deps:
	@echo "Installing development dependencies..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/goreleaser/goreleaser@latest

# Create a release with GoReleaser
release:
	@echo "Creating release with GoReleaser..."
	@goreleaser release --clean

# Create a snapshot release with GoReleaser (for testing)
release-snapshot:
	@echo "Creating snapshot release with GoReleaser..."
	@goreleaser release --snapshot --clean

# Help command
help:
	@echo "Available targets:"
	@echo "  all             - Run lint, test, and build"
	@echo "  build           - Build the application"
	@echo "  clean           - Remove build artifacts"
	@echo "  test            - Run tests"
	@echo "  lint            - Run linter"
	@echo "  lint-fix        - Fix linting issues automatically where possible"
	@echo "  dev-deps        - Install development dependencies"
	@echo "  release         - Create a release with GoReleaser"
	@echo "  release-snapshot - Create a snapshot release with GoReleaser (for testing)"
	@echo "  help            - Show this help message" 