package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flemzord/webhook-proxy/internal/config"
	"github.com/flemzord/webhook-proxy/internal/proxy"
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

// TestReadRequestBodyError tests the readRequestBody function with an error
func TestReadRequestBodyError(t *testing.T) {
	// Create a request with a body that will fail to read
	req := httptest.NewRequest(http.MethodPost, "/test", &MockReadCloser{})

	// Read the body
	body, err := readRequestBody(req)

	// Check for errors
	assert.Error(t, err)
	assert.Nil(t, body)
	assert.Contains(t, err.Error(), "mock read error")
}

// TestRegisterEndpointBodyReadError tests the registerEndpoint function with a body read error
func TestRegisterEndpointBodyReadError(t *testing.T) {
	// Create a minimal config with one endpoint
	cfg := &config.Config{
		Endpoints: []config.EndpointConfig{
			{
				Path: "/webhook-error",
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

	// Create a test request with a body that will fail to read
	req := httptest.NewRequest(http.MethodPost, "/webhook-error", &MockReadCloser{})
	w := httptest.NewRecorder()

	// Serve the request
	server.router.ServeHTTP(w, req)

	// Check the response
	resp := w.Result()
	defer resp.Body.Close()

	// Assert status code (should be 500 Internal Server Error)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(respBody), "Failed to read request body")
}

// TestRegisterMetricsEndpointEncodeError tests the registerMetricsEndpoint function with a JSON encode error
func TestRegisterMetricsEndpointEncodeError(t *testing.T) {
	// Create a minimal server
	cfg := &config.Config{}
	log := logrus.New()
	log.SetOutput(io.Discard) // Silence logs during tests
	server := NewServer(cfg, log)

	// Create a real proxy handler
	destinations := []config.DestinationConfig{
		{
			URL:     "http://example.com",
			Method:  "POST",
			Timeout: 5 * time.Second,
		},
	}
	handler := proxy.NewProxyHandler(destinations, log)

	// Add the handler to the server
	server.proxyHandlers["/test"] = handler

	// Register the metrics endpoint
	server.registerMetricsEndpoint()

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := &mockResponseWriter{
		header:     http.Header{},
		statusCode: http.StatusOK,
		writeError: true, // Simulate a write error
	}

	// Serve the request
	server.router.ServeHTTP(w, req)

	// Assert that the status code was set to 500 Internal Server Error
	assert.Equal(t, http.StatusInternalServerError, w.statusCode)
}

// TestRegisterMetricsResetEndpoint tests the metrics reset endpoint
func TestRegisterMetricsResetEndpoint(t *testing.T) {
	// Create a minimal server
	cfg := &config.Config{}
	log := logrus.New()
	log.SetOutput(io.Discard) // Silence logs during tests
	server := NewServer(cfg, log)

	// Create a real proxy handler
	destinations := []config.DestinationConfig{
		{
			URL:     "http://example.com",
			Method:  "POST",
			Timeout: 5 * time.Second,
		},
	}
	handler := proxy.NewProxyHandler(destinations, log)

	// Add the handler to the server
	server.proxyHandlers["/test"] = handler

	// Record some metrics
	handler.GetMetrics() // Initialize metrics

	// Register the metrics endpoint
	server.registerMetricsEndpoint()

	// Create a test request for metrics reset
	req := httptest.NewRequest(http.MethodPost, "/metrics/reset", nil)
	w := httptest.NewRecorder()

	// Serve the request
	server.router.ServeHTTP(w, req)

	// Check the response
	resp := w.Result()
	defer resp.Body.Close()

	// Assert status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(respBody), "Metrics reset successfully")

	// Verify metrics were reset by checking the metrics endpoint
	reqMetrics := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	wMetrics := httptest.NewRecorder()

	// Serve the request
	server.router.ServeHTTP(wMetrics, reqMetrics)

	// Check the response
	respMetrics := wMetrics.Result()
	defer respMetrics.Body.Close()

	// Read the response body
	respBodyMetrics, err := io.ReadAll(respMetrics.Body)
	assert.NoError(t, err)

	// Parse the metrics
	var metricsData map[string]interface{}
	err = json.Unmarshal(respBodyMetrics, &metricsData)
	assert.NoError(t, err)

	// Check that global metrics are reset
	global, ok := metricsData["global"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(0), global["total_requests"])
	assert.Equal(t, float64(0), global["successful_requests"])
	assert.Equal(t, float64(0), global["failed_requests"])
}

// TestRegisterHealthCheckEndpointEncodeError tests the registerHealthCheckEndpoint function with a JSON encode error
func TestRegisterHealthCheckEndpointEncodeError(t *testing.T) {
	// Create a minimal server
	cfg := &config.Config{}
	log := logrus.New()
	log.SetOutput(io.Discard) // Silence logs during tests
	server := NewServer(cfg, log)

	// Register the health check endpoint
	server.registerHealthCheckEndpoint()

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := &mockResponseWriter{
		header:     http.Header{},
		statusCode: http.StatusOK,
		writeError: true, // Simulate a write error
	}

	// Serve the request
	server.router.ServeHTTP(w, req)

	// Assert that the status code was set to 500 Internal Server Error
	assert.Equal(t, http.StatusInternalServerError, w.statusCode)
}

// mockResponseWriter is a mock implementation of http.ResponseWriter that can simulate write errors
type mockResponseWriter struct {
	header     http.Header
	statusCode int
	writeError bool
}

func (m *mockResponseWriter) Header() http.Header {
	return m.header
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	if m.writeError {
		return 0, fmt.Errorf("simulated write error")
	}
	return len(b), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

// TestStartServer tests the Start function
func TestStartServer(t *testing.T) {
	// Skip this test in normal test runs to avoid hanging
	t.Skip("Skipping test that starts a real server")

	// Create a minimal config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0, // Use port 0 to let the OS choose an available port
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

	// Start the server in a goroutine
	go func() {
		err := server.Start()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server.Start() error = %v", err)
		}
	}()

	// Wait a moment for the server to start
	time.Sleep(100 * time.Millisecond)

	// The test passes if we get here without panicking
	assert.NotNil(t, server)
}

// TestStartServerWithListener tests the Start function with a custom listener
func TestStartServerWithListener(t *testing.T) {
	// Skip this test in normal test runs to avoid hanging
	t.Skip("Skipping test that starts a real server")

	// Create a minimal config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0, // Use port 0 to let the OS choose an available port
		},
		Endpoints: []config.EndpointConfig{},
	}

	// Create a logger
	log := logrus.New()
	log.SetOutput(io.Discard) // Silence logs during tests

	// Create a new server
	server := NewServer(cfg, log)

	// Create a listener
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Get the port that was assigned
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("Failed to get TCP address from listener")
	}
	port := addr.Port
	cfg.Server.Port = port

	// Close the listener to simulate a port already in use
	listener.Close()

	// Try to start the server
	err = server.Start()

	// The error should not be nil (since the port is already in use)
	assert.Error(t, err)
}

// TestStartWithoutBlocking tests the Start function without actually starting the server
func TestStartWithoutBlocking(t *testing.T) {
	// Create a minimal config
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
						Method:  "POST",
						Timeout: 5 * time.Second,
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

	// Create a mock server function
	var capturedAddr string
	var capturedHandler http.Handler
	mockServerFunc := func(addr string, handler http.Handler) error {
		capturedAddr = addr
		capturedHandler = handler
		return nil // Return nil to simulate successful server start
	}

	// Call StartWithServerFunc
	err := server.StartWithServerFunc(mockServerFunc)

	// Verify that no error was returned
	assert.NoError(t, err)

	// Verify that the server function was called with the correct parameters
	assert.Equal(t, "localhost:8080", capturedAddr)
	assert.Equal(t, server.router, capturedHandler)

	// Verify that all endpoints were registered
	assert.Contains(t, server.proxyHandlers, "/webhook")
}

// MockReadCloser is a mock for io.ReadCloser that returns an error on Read
type MockReadCloser struct {
	io.Reader
}

func (m MockReadCloser) Close() error {
	return nil
}

func (m MockReadCloser) Read(_ []byte) (n int, err error) {
	return 0, errors.New("mock read error")
}

// TestRegisterMetricsResetEndpointWriteError tests the metrics reset endpoint with a write error
func TestRegisterMetricsResetEndpointWriteError(t *testing.T) {
	// Create a minimal server
	cfg := &config.Config{}
	log := logrus.New()
	log.SetOutput(io.Discard) // Silence logs during tests
	server := NewServer(cfg, log)

	// Create a real proxy handler
	destinations := []config.DestinationConfig{
		{
			URL:     "http://example.com",
			Method:  "POST",
			Timeout: 5 * time.Second,
		},
	}
	handler := proxy.NewProxyHandler(destinations, log)

	// Add the handler to the server
	server.proxyHandlers["/test"] = handler

	// Register the metrics endpoint
	server.registerMetricsEndpoint()

	// Create a test request for metrics reset
	req := httptest.NewRequest(http.MethodPost, "/metrics/reset", nil)
	w := &mockResponseWriter{
		header:     http.Header{},
		statusCode: http.StatusOK,
		writeError: true, // Simulate a write error
	}

	// Serve the request
	server.router.ServeHTTP(w, req)

	// Assert that the status code was set to 200 OK (the error is logged but doesn't change the status)
	assert.Equal(t, http.StatusOK, w.statusCode)
}

// TestStartWithError tests the Start function when the server function returns an error
func TestStartWithError(t *testing.T) {
	// Create a minimal config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	// Create a logger
	log := logrus.New()
	log.SetOutput(io.Discard) // Silence logs during tests

	// Create a new server
	server := NewServer(cfg, log)

	// Create a mock server function that returns an error
	expectedError := errors.New("failed to start server")
	mockServerFunc := func(_ string, _ http.Handler) error {
		return expectedError
	}

	// Call StartWithServerFunc
	err := server.StartWithServerFunc(mockServerFunc)

	// Verify that the expected error was returned
	assert.Equal(t, expectedError, err)
}

// TestStart tests the Start function with the default HTTP server function
func TestStart(t *testing.T) {
	// Create a minimal config
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
						Method:  "POST",
						Timeout: 5 * time.Second,
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

	// Save the original DefaultHTTPServerFunc and restore it after the test
	originalFunc := DefaultHTTPServerFunc
	defer func() {
		DefaultHTTPServerFunc = originalFunc
	}()

	// Replace DefaultHTTPServerFunc with a mock
	var capturedAddr string
	var capturedHandler http.Handler
	DefaultHTTPServerFunc = func(addr string, handler http.Handler) error {
		capturedAddr = addr
		capturedHandler = handler
		return nil
	}

	// Call Start
	err := server.Start()

	// Verify that no error was returned
	assert.NoError(t, err)

	// Verify that DefaultHTTPServerFunc was called with the correct parameters
	assert.Equal(t, "localhost:8080", capturedAddr)
	assert.Equal(t, server.router, capturedHandler)

	// Verify that all endpoints were registered
	assert.Contains(t, server.proxyHandlers, "/webhook")
}
