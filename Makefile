.PHONY: all build clean test lint lint-fix help release release-snapshot coverage

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
	@rm -f coverage.out

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Generate coverage report
coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./internal/...
	@go tool cover -func=coverage.out
	@echo "Total coverage:" 
	@go tool cover -func=coverage.out | grep total | awk '{print $$3}'
	@rm -f coverage.out

# Run linter and check test coverage
lint:
	@echo "Running linter..."
	@golangci-lint run ./...
	@echo "Checking test coverage..."
	@go test -coverprofile=coverage.out ./internal/...
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'
	@rm -f coverage.out

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
	@echo "  coverage        - Generate detailed coverage report"
	@echo "  lint            - Run linter and check test coverage"
	@echo "  lint-fix        - Fix linting issues automatically where possible"
	@echo "  dev-deps        - Install development dependencies"
	@echo "  release         - Create a release with GoReleaser"
	@echo "  release-snapshot - Create a snapshot release with GoReleaser (for testing)"
	@echo "  help            - Show this help message" 