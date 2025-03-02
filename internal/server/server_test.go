package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flemzord/webhook-proxy/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	// Create a minimal config for testing
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Endpoints: []config.EndpointConfig{},
	}

	// Create a logger
	log := logrus.New()
	log.SetOutput(io.Discard) // Silence logs during tests

	// Create a new server
	server := NewServer(cfg, log)

	// Assert that the server was created correctly
	assert.NotNil(t, server)
	assert.Equal(t, cfg, server.config)
	assert.NotNil(t, server.router)
	assert.Equal(t, log, server.log)
	assert.NotNil(t, server.proxyHandlers)
	assert.Equal(t, "1.0.0", server.version)
}

func TestSetVersion(t *testing.T) {
	// Create a minimal server
	server := &Server{
		version: "1.0.0",
	}

	// Set a new version
	newVersion := "2.0.0"
	server.SetVersion(newVersion)

	// Assert that the version was updated
	assert.Equal(t, newVersion, server.version)
}

func TestCalculateSuccessRate(t *testing.T) {
	tests := []struct {
		name       string
		successful int64
		total      int64
		expected   float64
	}{
		{
			name:       "Zero total",
			successful: 0,
			total:      0,
			expected:   0,
		},
		{
			name:       "All successful",
			successful: 10,
			total:      10,
			expected:   100,
		},
		{
			name:       "Half successful",
			successful: 5,
			total:      10,
			expected:   50,
		},
		{
			name:       "None successful",
			successful: 0,
			total:      10,
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSuccessRate(tt.successful, tt.total)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadRequestBody(t *testing.T) {
	// Test cases
	tests := []struct {
		name        string
		requestBody string
		expectError bool
	}{
		{
			name:        "Empty body",
			requestBody: "",
			expectError: false,
		},
		{
			name:        "Valid JSON body",
			requestBody: `{"key": "value"}`,
			expectError: false,
		},
		{
			name:        "Plain text body",
			requestBody: "Hello, world!",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with the test body
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.requestBody))

			// Read the body
			body, err := readRequestBody(req)

			// Check for errors
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.requestBody, string(body))
			}
		})
	}
}

func TestRegisterHealthCheckEndpoint(t *testing.T) {
	// Create a minimal server
	cfg := &config.Config{}
	log := logrus.New()
	log.SetOutput(io.Discard) // Silence logs during tests
	server := NewServer(cfg, log)

	// Set a specific version for testing
	testVersion := "test-version"
	server.SetVersion(testVersion)

	// Register the health check endpoint
	server.registerHealthCheckEndpoint()

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Serve the request
	server.router.ServeHTTP(w, req)

	// Check the response
	resp := w.Result()
	defer resp.Body.Close()

	// Assert status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Assert content type
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Parse the response body
	var health map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&health)
	assert.NoError(t, err)

	// Assert the response fields
	assert.Equal(t, "ok", health["status"])
	assert.Equal(t, testVersion, health["version"])
	assert.NotEmpty(t, health["timestamp"])
}

func TestRegisterEndpoint(t *testing.T) {
	// Create a minimal config with one endpoint
	cfg := &config.Config{
		Endpoints: []config.EndpointConfig{
			{
				Path: "/webhook",
				Destinations: []config.DestinationConfig{
					{
						URL:     "http://example.com",
						Timeout: 5,
						Retries: 3,
					},
				},
			},
		},
	}

	// Create a logger
	log := logrus.New()
	log.SetOutput(io.Discard) // Silence logs during tests

	// Create a new server
	server := NewServer(cfg, log)

	// Register the endpoint
	server.registerEndpoint(cfg.Endpoints[0])

	// Assert that the proxy handler was created
	assert.Contains(t, server.proxyHandlers, "/webhook")
	assert.NotNil(t, server.proxyHandlers["/webhook"])

	// Create a test request
	body := []byte(`{"test": "data"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Serve the request
	server.router.ServeHTTP(w, req)

	// Check the response
	resp := w.Result()
	defer resp.Body.Close()

	// Assert status code (should be 200 OK as we return immediately)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRegisterMetricsEndpoint(t *testing.T) {
	// Create a minimal server
	cfg := &config.Config{}
	log := logrus.New()
	log.SetOutput(io.Discard) // Silence logs during tests
	server := NewServer(cfg, log)

	// Register the metrics endpoint
	server.registerMetricsEndpoint()

	// Create a test request for metrics
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	// Serve the request
	server.router.ServeHTTP(w, req)

	// Check the response
	resp := w.Result()
	defer resp.Body.Close()

	// Assert status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Assert content type
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Parse the response body
	var metrics map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&metrics)
	assert.NoError(t, err)

	// Assert the response structure
	assert.Contains(t, metrics, "global")
	assert.Contains(t, metrics, "endpoints")
	assert.Contains(t, metrics, "timestamp")

	// Test metrics reset endpoint
	resetReq := httptest.NewRequest(http.MethodPost, "/metrics/reset", nil)
	resetW := httptest.NewRecorder()

	// Serve the request
	server.router.ServeHTTP(resetW, resetReq)

	// Check the response
	resetResp := resetW.Result()
	defer resetResp.Body.Close()

	// Assert status code
	assert.Equal(t, http.StatusOK, resetResp.StatusCode)

	// Parse the response body
	var resetResponse map[string]interface{}
	err = json.NewDecoder(resetResp.Body).Decode(&resetResponse)
	assert.NoError(t, err)

	// Assert the response fields
	assert.Equal(t, "ok", resetResponse["status"])
	assert.Equal(t, "Metrics reset successfully", resetResponse["message"])
}

func TestStartServerSetup(t *testing.T) {
	// Create a minimal config with one endpoint
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Endpoints: []config.EndpointConfig{
			{
				Path: "/webhook",
				Destinations: []config.DestinationConfig{
					{
						URL:     "http://example.com",
						Timeout: 5,
						Retries: 3,
					},
				},
			},
		},
	}

	// Create a logger
	log := logrus.New()
	log.SetOutput(io.Discard) // Silence logs during tests

	// Create a new server
	server := NewServer(cfg, log)

	// Create a test server that will be used instead of calling Start()
	// This allows us to test the setup without actually starting the server
	testServer := httptest.NewServer(server.router)
	defer testServer.Close()

	// Manually register the endpoints that Start() would register
	for _, endpoint := range cfg.Endpoints {
		server.registerEndpoint(endpoint)
	}
	server.registerMetricsEndpoint()
	server.registerHealthCheckEndpoint()

	// Verify that the endpoints were registered correctly
	assert.Contains(t, server.proxyHandlers, "/webhook")

	// Test that the webhook endpoint responds correctly
	resp, err := http.Post(testServer.URL+"/webhook", "application/json", bytes.NewReader([]byte(`{"test":"data"}`)))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Test that the metrics endpoint responds correctly
	resp, err = http.Get(testServer.URL + "/metrics")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Test that the health endpoint responds correctly
	resp, err = http.Get(testServer.URL + "/health")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}
