package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `
server:
  port: 8080
  host: "127.0.0.1"

logging:
  level: "info"
  format: "json"
  output: "stdout"

endpoints:
  - path: "/webhook/test"
    destinations:
      - url: "https://example.com/webhook"
        method: "POST"
        headers:
          Content-Type: "application/json"
        timeout: 5s
        retries: 3
        retry_delay: 1s
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Load the config
	config, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify server config
	if config.Server.Port != 8080 {
		t.Errorf("Expected server port 8080, got %d", config.Server.Port)
	}
	if config.Server.Host != "127.0.0.1" {
		t.Errorf("Expected server host 127.0.0.1, got %s", config.Server.Host)
	}

	// Verify logging config
	if config.Logging.Level != "info" {
		t.Errorf("Expected logging level info, got %s", config.Logging.Level)
	}
	if config.Logging.Format != "json" {
		t.Errorf("Expected logging format json, got %s", config.Logging.Format)
	}
	if config.Logging.Output != "stdout" {
		t.Errorf("Expected logging output stdout, got %s", config.Logging.Output)
	}

	// Verify endpoints
	if len(config.Endpoints) != 1 {
		t.Fatalf("Expected 1 endpoint, got %d", len(config.Endpoints))
	}
	if config.Endpoints[0].Path != "/webhook/test" {
		t.Errorf("Expected endpoint path /webhook/test, got %s", config.Endpoints[0].Path)
	}

	// Verify destinations
	if len(config.Endpoints[0].Destinations) != 1 {
		t.Fatalf("Expected 1 destination, got %d", len(config.Endpoints[0].Destinations))
	}
	dest := config.Endpoints[0].Destinations[0]
	if dest.URL != "https://example.com/webhook" {
		t.Errorf("Expected destination URL https://example.com/webhook, got %s", dest.URL)
	}
	if dest.Method != "POST" {
		t.Errorf("Expected destination method POST, got %s", dest.Method)
	}
	if dest.Timeout != 5*time.Second {
		t.Errorf("Expected destination timeout 5s, got %s", dest.Timeout)
	}
	if dest.Retries != 3 {
		t.Errorf("Expected destination retries 3, got %d", dest.Retries)
	}
	if dest.RetryDelay != 1*time.Second {
		t.Errorf("Expected destination retry delay 1s, got %s", dest.RetryDelay)
	}
	if dest.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header application/json, got %s", dest.Headers["Content-Type"])
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Create a minimal config file to test defaults
	configContent := `
endpoints:
  - path: "/webhook/test"
    destinations:
      - url: "https://example.com/webhook"
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Load the config
	config, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify default server config
	if config.Server.Port != 8080 {
		t.Errorf("Expected default server port 8080, got %d", config.Server.Port)
	}
	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default server host 0.0.0.0, got %s", config.Server.Host)
	}

	// Verify default logging config
	if config.Logging.Level != "info" {
		t.Errorf("Expected default logging level info, got %s", config.Logging.Level)
	}
	if config.Logging.Format != "json" {
		t.Errorf("Expected default logging format json, got %s", config.Logging.Format)
	}
	if config.Logging.Output != "stdout" {
		t.Errorf("Expected default logging output stdout, got %s", config.Logging.Output)
	}

	// Verify default destination config
	dest := config.Endpoints[0].Destinations[0]
	if dest.Method != "POST" {
		t.Errorf("Expected default destination method POST, got %s", dest.Method)
	}
	if dest.Timeout != 5*time.Second {
		t.Errorf("Expected default destination timeout 5s, got %s", dest.Timeout)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "Valid config",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
				Endpoints: []EndpointConfig{
					{
						Path: "/webhook/test",
						Destinations: []DestinationConfig{
							{
								URL:    "https://example.com/webhook",
								Method: "POST",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Invalid server port",
			config: Config{
				Server: ServerConfig{
					Port: 70000, // Invalid port
					Host: "0.0.0.0",
				},
				Endpoints: []EndpointConfig{
					{
						Path: "/webhook/test",
						Destinations: []DestinationConfig{
							{
								URL:    "https://example.com/webhook",
								Method: "POST",
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "Invalid logging level",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				Logging: LoggingConfig{
					Level:  "invalid", // Invalid level
					Format: "json",
					Output: "stdout",
				},
				Endpoints: []EndpointConfig{
					{
						Path: "/webhook/test",
						Destinations: []DestinationConfig{
							{
								URL:    "https://example.com/webhook",
								Method: "POST",
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "Missing endpoint path",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
				Endpoints: []EndpointConfig{
					{
						Path: "", // Missing path
						Destinations: []DestinationConfig{
							{
								URL:    "https://example.com/webhook",
								Method: "POST",
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "Missing destination URL",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
				Endpoints: []EndpointConfig{
					{
						Path: "/webhook/test",
						Destinations: []DestinationConfig{
							{
								URL:    "", // Missing URL
								Method: "POST",
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "No destinations",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
				Endpoints: []EndpointConfig{
					{
						Path:         "/webhook/test",
						Destinations: []DestinationConfig{}, // No destinations
					},
				},
			},
			expectError: true,
		},
		{
			name: "File output without file path",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				Logging: LoggingConfig{
					Level:    "info",
					Format:   "json",
					Output:   "file",
					FilePath: "", // Missing file path
				},
				Endpoints: []EndpointConfig{
					{
						Path: "/webhook/test",
						Destinations: []DestinationConfig{
							{
								URL:    "https://example.com/webhook",
								Method: "POST",
							},
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.config)
			if (err != nil) != tt.expectError {
				t.Errorf("validateConfig() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
