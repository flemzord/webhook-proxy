package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Default configuration values
const (
	DefaultLogLevel  = "info"
	DefaultLogFormat = "json"
	DefaultLogOutput = "stdout"
	DefaultMethod    = "POST"
	DefaultHost      = "0.0.0.0"
)

// Config represents the application configuration
type Config struct {
	Server    ServerConfig     `yaml:"server"`
	Logging   LoggingConfig    `yaml:"logging"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
}

// ServerConfig represents the server configuration
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

// EndpointConfig represents an endpoint configuration
type EndpointConfig struct {
	Path         string              `yaml:"path"`
	Destinations []DestinationConfig `yaml:"destinations"`
}

// DestinationConfig represents a destination configuration
type DestinationConfig struct {
	URL        string            `yaml:"url"`
	Method     string            `yaml:"method"`
	Headers    map[string]string `yaml:"headers"`
	Timeout    time.Duration     `yaml:"timeout"`
	Retries    int               `yaml:"retries"`
	RetryDelay time.Duration     `yaml:"retry_delay"`
}

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	// Read the configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Parse the YAML
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Set default values
	setDefaultValues(&config)

	// Apply environment variable overrides
	applyEnvironmentOverrides(&config)

	// Validate the configuration
	err = validateConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
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
		config.Server.Host = DefaultHost
	}

	// Logging defaults
	if config.Logging.Level == "" {
		config.Logging.Level = DefaultLogLevel
	}
	if config.Logging.Format == "" {
		config.Logging.Format = DefaultLogFormat
	}
	if config.Logging.Output == "" {
		config.Logging.Output = DefaultLogOutput
	}

	// Endpoint defaults
	for i := range config.Endpoints {
		for j := range config.Endpoints[i].Destinations {
			dest := &config.Endpoints[i].Destinations[j]

			// Default method is POST
			if dest.Method == "" {
				dest.Method = DefaultMethod
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
			if dest.RetryDelay == 0 {
				dest.RetryDelay = 1 * time.Second
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
	if err := validateServerConfig(&config.Server); err != nil {
		return err
	}

	// Validate logging configuration
	if err := validateLoggingConfig(&config.Logging); err != nil {
		return err
	}

	// Validate endpoints
	if len(config.Endpoints) == 0 {
		return fmt.Errorf("at least one endpoint is required")
	}

	for i, endpoint := range config.Endpoints {
		if err := validateEndpointConfig(i, endpoint); err != nil {
			return err
		}
	}

	return nil
}

// validateServerConfig validates the server configuration
func validateServerConfig(server *ServerConfig) error {
	if server.Port < 0 || server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", server.Port)
	}
	return nil
}

// validateLoggingConfig validates the logging configuration
func validateLoggingConfig(logging *LoggingConfig) error {
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[logging.Level] {
		return fmt.Errorf("invalid logging level: %s", logging.Level)
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[logging.Format] {
		return fmt.Errorf("invalid logging format: %s", logging.Format)
	}

	validOutputs := map[string]bool{"stdout": true, "file": true}
	if !validOutputs[logging.Output] {
		return fmt.Errorf("invalid logging output: %s", logging.Output)
	}

	if logging.Output == "file" && logging.FilePath == "" {
		return fmt.Errorf("file_path is required when output is file")
	}

	return nil
}

// validateEndpointConfig validates an endpoint configuration
func validateEndpointConfig(index int, endpoint EndpointConfig) error {
	if endpoint.Path == "" {
		return fmt.Errorf("endpoint[%d]: path is required", index)
	}

	// Ensure path starts with /
	if !strings.HasPrefix(endpoint.Path, "/") {
		return fmt.Errorf("endpoint[%d]: path must start with /", index)
	}

	if len(endpoint.Destinations) == 0 {
		return fmt.Errorf("endpoint[%d]: at least one destination is required", index)
	}

	for j, dest := range endpoint.Destinations {
		if err := validateDestinationConfig(index, j, dest); err != nil {
			return err
		}
	}

	return nil
}

// validateDestinationConfig validates a destination configuration
func validateDestinationConfig(endpointIndex, destIndex int, dest DestinationConfig) error {
	if dest.URL == "" {
		return fmt.Errorf("endpoint[%d].destination[%d]: url is required", endpointIndex, destIndex)
	}

	// Validate URL
	_, err := url.ParseRequestURI(dest.URL)
	if err != nil {
		return fmt.Errorf("endpoint[%d].destination[%d]: invalid url: %s", endpointIndex, destIndex, err)
	}

	// Validate HTTP method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "HEAD": true, "OPTIONS": true,
	}
	if !validMethods[strings.ToUpper(dest.Method)] {
		return fmt.Errorf("endpoint[%d].destination[%d]: invalid method: %s", endpointIndex, destIndex, dest.Method)
	}

	// Validate timeout
	if dest.Timeout < 0 {
		return fmt.Errorf("endpoint[%d].destination[%d]: timeout cannot be negative", endpointIndex, destIndex)
	}

	// Validate retries
	if dest.Retries < 0 {
		return fmt.Errorf("endpoint[%d].destination[%d]: retries cannot be negative", endpointIndex, destIndex)
	}

	// Validate retry delay
	if dest.RetryDelay < 0 {
		return fmt.Errorf("endpoint[%d].destination[%d]: retry_delay cannot be negative", endpointIndex, destIndex)
	}

	return nil
}
