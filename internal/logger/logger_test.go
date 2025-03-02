package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/flemzord/webhook-proxy/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	// Create a new logger
	log := NewLogger()

	// Assert that the logger was created correctly
	assert.NotNil(t, log)
	assert.IsType(t, &logrus.Logger{}, log)
	assert.IsType(t, &logrus.JSONFormatter{}, log.Formatter)
	assert.Equal(t, logrus.InfoLevel, log.Level)
}

func TestConfigureLoggerLevel(t *testing.T) {
	// Test cases for log levels
	testCases := []struct {
		name          string
		level         string
		expectedLevel logrus.Level
	}{
		{
			name:          "Debug level",
			level:         "debug",
			expectedLevel: logrus.DebugLevel,
		},
		{
			name:          "Info level",
			level:         "info",
			expectedLevel: logrus.InfoLevel,
		},
		{
			name:          "Warn level",
			level:         "warn",
			expectedLevel: logrus.WarnLevel,
		},
		{
			name:          "Error level",
			level:         "error",
			expectedLevel: logrus.ErrorLevel,
		},
		{
			name:          "Invalid level defaults to Info",
			level:         "invalid",
			expectedLevel: logrus.InfoLevel,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a logger
			log := logrus.New()

			// Configure the logger
			cfg := config.LoggingConfig{
				Level:  tc.level,
				Format: "json",
				Output: "stdout",
			}
			ConfigureLogger(log, cfg)

			// Assert that the level was set correctly
			assert.Equal(t, tc.expectedLevel, log.Level)
		})
	}
}

func TestConfigureLoggerFormat(t *testing.T) {
	// Test cases for log formats
	testCases := []struct {
		name           string
		format         string
		expectedFormat interface{}
	}{
		{
			name:           "JSON format",
			format:         "json",
			expectedFormat: &logrus.JSONFormatter{},
		},
		{
			name:           "Text format",
			format:         "text",
			expectedFormat: &logrus.TextFormatter{},
		},
		{
			name:           "Invalid format defaults to JSON",
			format:         "invalid",
			expectedFormat: &logrus.JSONFormatter{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a logger
			log := logrus.New()

			// Configure the logger
			cfg := config.LoggingConfig{
				Level:  "info",
				Format: tc.format,
				Output: "stdout",
			}
			ConfigureLogger(log, cfg)

			// Assert that the formatter was set correctly
			assert.Equal(t, reflect.TypeOf(tc.expectedFormat), reflect.TypeOf(log.Formatter))
		})
	}
}

func TestConfigureLoggerOutput(t *testing.T) {
	// Test stdout output
	t.Run("Stdout output", func(t *testing.T) {
		// Create a logger
		log := logrus.New()

		// Configure the logger
		cfg := config.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}
		ConfigureLogger(log, cfg)

		// We can't directly check the output, but we can ensure no error occurs
		assert.NotNil(t, log.Out)
	})

	// Test file output with valid path
	t.Run("File output with valid path", func(t *testing.T) {
		// Create a temporary file
		tmpDir, err := os.MkdirTemp("", "logger-test")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		logPath := filepath.Join(tmpDir, "test.log")

		// Create a logger
		log := logrus.New()

		// Configure the logger
		cfg := config.LoggingConfig{
			Level:    "info",
			Format:   "json",
			Output:   "file",
			FilePath: logPath,
		}
		ConfigureLogger(log, cfg)

		// Write a log entry
		log.Info("Test log entry")

		// Check that the file was created
		_, err = os.Stat(logPath)
		assert.NoError(t, err)

		// Read the file content
		content, err := os.ReadFile(logPath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "Test log entry")
	})

	// Test file output with invalid path
	t.Run("File output with invalid path", func(t *testing.T) {
		// Create a logger with a custom output to capture logs
		log := logrus.New()
		var buf bytes.Buffer
		log.SetOutput(&buf)

		// Configure the logger with an invalid path
		cfg := config.LoggingConfig{
			Level:    "info",
			Format:   "json",
			Output:   "file",
			FilePath: "/nonexistent/directory/test.log",
		}

		// Redirect stderr temporarily to avoid polluting test output
		oldStderr := os.Stderr
		_, w, err := os.Pipe()
		assert.NoError(t, err)
		os.Stderr = w

		ConfigureLogger(log, cfg)

		// Restore stderr
		w.Close()
		os.Stderr = oldStderr

		// The logger should fall back to stdout
		assert.NotNil(t, log.Out)
	})

	// Test default output
	t.Run("Default output", func(t *testing.T) {
		// Create a logger
		log := logrus.New()

		// Configure the logger with an invalid output
		cfg := config.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "invalid",
		}
		ConfigureLogger(log, cfg)

		// The logger should default to stdout
		assert.NotNil(t, log.Out)
	})
}

func TestLogWebhookReceived(t *testing.T) {
	// Create a logger with a buffer to capture output
	log := logrus.New()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.JSONFormatter{})

	// Call the function
	path := "/webhook"
	method := "POST"
	remoteAddr := "127.0.0.1"
	contentLength := int64(100)

	LogWebhookReceived(log, path, method, remoteAddr, contentLength)

	// Parse the log output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	// Assert that the log entry contains the expected fields
	assert.Equal(t, "Webhook received", logEntry["msg"])
	assert.Equal(t, path, logEntry["path"])
	assert.Equal(t, method, logEntry["method"])
	assert.Equal(t, remoteAddr, logEntry["remote_addr"])
	assert.Equal(t, float64(contentLength), logEntry["content_length"])
	assert.Equal(t, "info", logEntry["level"])
}

func TestLogWebhookForwarded(t *testing.T) {
	// Create a logger with a buffer to capture output
	log := logrus.New()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.JSONFormatter{})

	// Call the function
	destination := "http://example.com"
	statusCode := 200
	duration := 123.45

	LogWebhookForwarded(log, destination, statusCode, duration)

	// Parse the log output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	// Assert that the log entry contains the expected fields
	assert.Equal(t, "Webhook forwarded", logEntry["msg"])
	assert.Equal(t, destination, logEntry["destination"])
	assert.Equal(t, float64(statusCode), logEntry["status_code"])
	assert.Equal(t, duration, logEntry["duration_ms"])
	assert.Equal(t, "info", logEntry["level"])
}

func TestLogWebhookError(t *testing.T) {
	// Create a logger with a buffer to capture output
	log := logrus.New()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.JSONFormatter{})

	// Call the function
	destination := "http://example.com"
	err := errors.New("test error")
	attempt := 2
	maxAttempts := 3

	LogWebhookError(log, destination, err, attempt, maxAttempts)

	// Parse the log output
	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	// Assert that the log entry contains the expected fields
	assert.Equal(t, "Webhook forwarding failed", logEntry["msg"])
	assert.Equal(t, destination, logEntry["destination"])
	assert.Equal(t, "test error", logEntry["error"])
	assert.Equal(t, float64(attempt), logEntry["attempt"])
	assert.Equal(t, float64(maxAttempts), logEntry["max_attempts"])
	assert.Equal(t, "error", logEntry["level"])
}
