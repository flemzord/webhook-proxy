package config

import (
	"os"
	"testing"
)

// Define a constant for the test configuration content
const testConfigContent = `
endpoints:
  - path: "/webhook/test"
    destinations:
      - url: "https://example.com/webhook"
`

func TestEnvironmentOverrides(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, writeErr := tmpfile.Write([]byte(testConfigContent)); writeErr != nil {
		t.Fatalf("Failed to write to temp file: %v", writeErr)
	}
	if closeErr := tmpfile.Close(); closeErr != nil {
		t.Fatalf("Failed to close temp file: %v", closeErr)
	}

	// Set environment variables
	os.Setenv("WEBHOOK_PROXY_SERVER_PORT", "9090")
	os.Setenv("WEBHOOK_PROXY_SERVER_HOST", "127.0.0.1")
	os.Setenv("WEBHOOK_PROXY_LOG_LEVEL", "debug")
	os.Setenv("WEBHOOK_PROXY_LOG_FORMAT", "text")
	os.Setenv("WEBHOOK_PROXY_LOG_OUTPUT", "file")
	os.Setenv("WEBHOOK_PROXY_LOG_FILE_PATH", "/var/log/webhook-proxy.log")
	defer func() {
		os.Unsetenv("WEBHOOK_PROXY_SERVER_PORT")
		os.Unsetenv("WEBHOOK_PROXY_SERVER_HOST")
		os.Unsetenv("WEBHOOK_PROXY_LOG_LEVEL")
		os.Unsetenv("WEBHOOK_PROXY_LOG_FORMAT")
		os.Unsetenv("WEBHOOK_PROXY_LOG_OUTPUT")
		os.Unsetenv("WEBHOOK_PROXY_LOG_FILE_PATH")
	}()

	// Load the config
	config, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment variables were applied
	if config.Server.Port != 9090 {
		t.Errorf("Expected server port 9090, got %d", config.Server.Port)
	}
	if config.Server.Host != "127.0.0.1" {
		t.Errorf("Expected server host 127.0.0.1, got %s", config.Server.Host)
	}
	if config.Logging.Level != "debug" {
		t.Errorf("Expected logging level debug, got %s", config.Logging.Level)
	}
	if config.Logging.Format != "text" {
		t.Errorf("Expected logging format text, got %s", config.Logging.Format)
	}
	if config.Logging.Output != "file" {
		t.Errorf("Expected logging output file, got %s", config.Logging.Output)
	}
	if config.Logging.FilePath != "/var/log/webhook-proxy.log" {
		t.Errorf("Expected logging file path /var/log/webhook-proxy.log, got %s", config.Logging.FilePath)
	}
}

func TestInvalidEnvironmentOverrides(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, writeErr := tmpfile.Write([]byte(testConfigContent)); writeErr != nil {
		t.Fatalf("Failed to write to temp file: %v", writeErr)
	}
	if closeErr := tmpfile.Close(); closeErr != nil {
		t.Fatalf("Failed to close temp file: %v", closeErr)
	}

	// Set invalid environment variables
	os.Setenv("WEBHOOK_PROXY_SERVER_PORT", "invalid")
	os.Setenv("WEBHOOK_PROXY_LOG_LEVEL", "invalid")
	defer func() {
		os.Unsetenv("WEBHOOK_PROXY_SERVER_PORT")
		os.Unsetenv("WEBHOOK_PROXY_LOG_LEVEL")
	}()

	// Load the config - should fail due to invalid port
	_, loadErr := LoadConfig(tmpfile.Name())
	if loadErr == nil {
		t.Fatalf("Expected error due to invalid port, but got nil")
	}

	// Set valid port but invalid log level
	os.Setenv("WEBHOOK_PROXY_SERVER_PORT", "9090")
	_, loadErr = LoadConfig(tmpfile.Name())
	if loadErr == nil {
		t.Fatalf("Expected error due to invalid log level, but got nil")
	}
}
