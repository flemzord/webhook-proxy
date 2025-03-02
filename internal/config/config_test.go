package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	configContent := `
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
      - url: "https://example.com/webhook"
        method: "POST"
        headers:
          Content-Type: "application/json"
          X-Custom-Header: "custom-value"
        timeout: 10s
        retries: 3
        retry_delay: 1s
`
	tmpFileName := createTempConfigFile(t, configContent)
	defer os.Remove(tmpFileName)

	// Load the config
	config, err := LoadConfig(tmpFileName)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify server config
	if config.Server.Port != 8080 {
		t.Errorf("Expected server port 8080, got %d", config.Server.Port)
	}
	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Expected server host 0.0.0.0, got %s", config.Server.Host)
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
	if config.Endpoints[0].Path != "/webhook/github" {
		t.Errorf("Expected endpoint path /webhook/github, got %s", config.Endpoints[0].Path)
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
	if dest.Timeout != 10*time.Second {
		t.Errorf("Expected destination timeout 10s, got %s", dest.Timeout)
	}
	if dest.Retries != 3 {
		t.Errorf("Expected destination retries 3, got %d", dest.Retries)
	}
	if dest.RetryDelay != 1*time.Second {
		t.Errorf("Expected destination retry delay 1s, got %s", dest.RetryDelay)
	}
	if len(dest.Headers) != 2 {
		t.Fatalf("Expected 2 headers, got %d", len(dest.Headers))
	}
	if dest.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header application/json, got %s", dest.Headers["Content-Type"])
	}
	if dest.Headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("Expected X-Custom-Header header custom-value, got %s", dest.Headers["X-Custom-Header"])
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("non-existent-file.yaml")
	if err == nil {
		t.Fatalf("Expected error for non-existent file, got nil")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	invalidYAML := `
server:
  port: 8080
  host: "0.0.0.0"
  invalid yaml content
`
	tmpFileName := createTempConfigFile(t, invalidYAML)
	defer os.Remove(tmpFileName)

	_, err := LoadConfig(tmpFileName)
	if err == nil {
		t.Fatalf("Expected error for invalid YAML, got nil")
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
	tmpFileName := createTempConfigFile(t, configContent)
	defer os.Remove(tmpFileName)

	// Load the config
	config, err := LoadConfig(tmpFileName)
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

func TestSetDefaultValuesWithNegativeRetries(t *testing.T) {
	config := &Config{
		Endpoints: []EndpointConfig{
			{
				Path: "/webhook/test",
				Destinations: []DestinationConfig{
					{
						URL:     "https://example.com/webhook",
						Retries: -1, // Negative retries should be set to 0
					},
				},
			},
		},
	}

	setDefaultValues(config)

	if config.Endpoints[0].Destinations[0].Retries != 0 {
		t.Errorf("Expected retries to be set to 0, got %d", config.Endpoints[0].Destinations[0].Retries)
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
			name: "Invalid logging format",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "invalid", // Invalid format
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
			name: "Invalid logging output",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "0.0.0.0",
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "invalid", // Invalid output
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
			name: "Path without leading slash",
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
						Path: "webhook/test", // Missing leading slash
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
			name: "Invalid destination URL",
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
								URL:    "invalid-url", // Invalid URL
								Method: "POST",
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "Invalid HTTP method",
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
								Method: "INVALID", // Invalid method
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "Negative timeout",
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
								URL:     "https://example.com/webhook",
								Method:  "POST",
								Timeout: -1 * time.Second, // Negative timeout
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "Negative retries",
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
								URL:     "https://example.com/webhook",
								Method:  "POST",
								Retries: -1, // Negative retries
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "Negative retry delay",
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
								URL:        "https://example.com/webhook",
								Method:     "POST",
								RetryDelay: -1 * time.Second, // Negative retry delay
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
			name: "No endpoints",
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
				Endpoints: []EndpointConfig{}, // No endpoints
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

// Helper function to create a temporary config file
func createTempConfigFile(t *testing.T, configContent string) string {
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, writeErr := tmpfile.Write([]byte(configContent)); writeErr != nil {
		t.Fatalf("Failed to write to temp file: %v", writeErr)
	}

	if closeErr := tmpfile.Close(); closeErr != nil {
		t.Fatalf("Failed to close temp file: %v", closeErr)
	}

	return tmpfile.Name()
}
