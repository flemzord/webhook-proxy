package config

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server    ServerConfig     `yaml:"server"`
	Logging   LoggingConfig    `yaml:"logging"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
}

// ServerConfig represents the HTTP server configuration
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// LoggingConfig represents the logging configuration
type LoggingConfig struct {
	Level    string `yaml:"level"`
	Format   string `yaml:"format"`
	Output   string `yaml:"output"`
	FilePath string `yaml:"file_path"`
}

// EndpointConfig represents a webhook endpoint configuration
type EndpointConfig struct {
	Path         string              `yaml:"path"`
	Destinations []DestinationConfig `yaml:"destinations"`
}

// DestinationConfig represents a webhook destination configuration
type DestinationConfig struct {
	URL        string            `yaml:"url"`
	Method     string            `yaml:"method"`
	Headers    map[string]string `yaml:"headers"`
	Timeout    time.Duration     `yaml:"timeout"`
	Retries    int               `yaml:"retries"`
	RetryDelay time.Duration     `yaml:"retry_delay"`
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", path)
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	applyEnvironmentOverrides(&config)

	// Set default values
	setDefaultValues(&config)

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaultValues sets default values for the configuration
func setDefaultValues(config *Config) {
	// Server defaults
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}

	// Logging defaults
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}
	if config.Logging.Output == "" {
		config.Logging.Output = "stdout"
	}

	// Endpoint defaults
	for i := range config.Endpoints {
		for j := range config.Endpoints[i].Destinations {
			dest := &config.Endpoints[i].Destinations[j]

			// Default method is POST
			if dest.Method == "" {
				dest.Method = "POST"
			}

			// Default timeout is 5 seconds
			if dest.Timeout == 0 {
				dest.Timeout = 5 * time.Second
			}

			// Default retries is 0 (no retries)
			if dest.Retries < 0 {
				dest.Retries = 0
			}

			// Default retry delay is 1 second
			if dest.RetryDelay == 0 && dest.Retries > 0 {
				dest.RetryDelay = 1 * time.Second
			}

			// Initialize headers map if nil
			if dest.Headers == nil {
				dest.Headers = make(map[string]string)
			}
		}
	}
}

// applyEnvironmentOverrides applies environment variable overrides to the configuration
func applyEnvironmentOverrides(config *Config) {
	// Server overrides
	if port, exists := os.LookupEnv("WEBHOOK_PROXY_SERVER_PORT"); exists {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}
	if host, exists := os.LookupEnv("WEBHOOK_PROXY_SERVER_HOST"); exists {
		config.Server.Host = host
	}

	// Logging overrides
	if level, exists := os.LookupEnv("WEBHOOK_PROXY_LOG_LEVEL"); exists {
		config.Logging.Level = level
	}
	if format, exists := os.LookupEnv("WEBHOOK_PROXY_LOG_FORMAT"); exists {
		config.Logging.Format = format
	}
	if output, exists := os.LookupEnv("WEBHOOK_PROXY_LOG_OUTPUT"); exists {
		config.Logging.Output = output
	}
	if filePath, exists := os.LookupEnv("WEBHOOK_PROXY_LOG_FILE_PATH"); exists {
		config.Logging.FilePath = filePath
	}

	// Endpoints cannot be easily overridden with environment variables
	// due to their complex structure. Use the YAML file for endpoints.
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate server configuration
	if config.Server.Port < 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	// Validate logging configuration
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[config.Logging.Level] {
		return fmt.Errorf("invalid logging level: %s", config.Logging.Level)
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[config.Logging.Format] {
		return fmt.Errorf("invalid logging format: %s", config.Logging.Format)
	}

	validOutputs := map[string]bool{"stdout": true, "file": true}
	if !validOutputs[config.Logging.Output] {
		return fmt.Errorf("invalid logging output: %s", config.Logging.Output)
	}

	if config.Logging.Output == "file" && config.Logging.FilePath == "" {
		return fmt.Errorf("file_path is required when output is file")
	}

	// Validate endpoints
	if len(config.Endpoints) == 0 {
		return fmt.Errorf("at least one endpoint is required")
	}

	for i, endpoint := range config.Endpoints {
		if endpoint.Path == "" {
			return fmt.Errorf("endpoint[%d]: path is required", i)
		}

		// Ensure path starts with /
		if !strings.HasPrefix(endpoint.Path, "/") {
			return fmt.Errorf("endpoint[%d]: path must start with /", i)
		}

		if len(endpoint.Destinations) == 0 {
			return fmt.Errorf("endpoint[%d]: at least one destination is required", i)
		}

		for j, dest := range endpoint.Destinations {
			if dest.URL == "" {
				return fmt.Errorf("endpoint[%d].destination[%d]: url is required", i, j)
			}

			// Validate URL
			_, err := url.ParseRequestURI(dest.URL)
			if err != nil {
				return fmt.Errorf("endpoint[%d].destination[%d]: invalid url: %s", i, j, err)
			}

			// Validate HTTP method
			validMethods := map[string]bool{
				"GET": true, "POST": true, "PUT": true, "DELETE": true,
				"PATCH": true, "HEAD": true, "OPTIONS": true,
			}
			if !validMethods[strings.ToUpper(dest.Method)] {
				return fmt.Errorf("endpoint[%d].destination[%d]: invalid method: %s", i, j, dest.Method)
			}

			// Validate timeout
			if dest.Timeout < 0 {
				return fmt.Errorf("endpoint[%d].destination[%d]: timeout cannot be negative", i, j)
			}

			// Validate retries
			if dest.Retries < 0 {
				return fmt.Errorf("endpoint[%d].destination[%d]: retries cannot be negative", i, j)
			}

			// Validate retry delay
			if dest.RetryDelay < 0 {
				return fmt.Errorf("endpoint[%d].destination[%d]: retry_delay cannot be negative", i, j)
			}
		}
	}

	return nil
}
