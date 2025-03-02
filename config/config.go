package config

import (
	"fmt"
	"io/ioutil"
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
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default values
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}
	if config.Logging.Output == "" {
		config.Logging.Output = "stdout"
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
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
	for i, endpoint := range config.Endpoints {
		if endpoint.Path == "" {
			return fmt.Errorf("endpoint[%d]: path is required", i)
		}

		if len(endpoint.Destinations) == 0 {
			return fmt.Errorf("endpoint[%d]: at least one destination is required", i)
		}

		for j, dest := range endpoint.Destinations {
			if dest.URL == "" {
				return fmt.Errorf("endpoint[%d].destination[%d]: url is required", i, j)
			}

			if dest.Method == "" {
				// Default to POST if not specified
				config.Endpoints[i].Destinations[j].Method = "POST"
			}

			// Set default timeout if not specified
			if dest.Timeout == 0 {
				config.Endpoints[i].Destinations[j].Timeout = 5 * time.Second
			}
		}
	}

	return nil
}
