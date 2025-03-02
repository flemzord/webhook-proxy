package proxy

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flemzord/webhook-proxy/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestProxyHandler_ForwardWebhook(t *testing.T) {
	// Create test servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))

		// Return success
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok"}`))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate an error
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(`{"status":"error"}`))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server2.Close()

	// Create destinations
	destinations := []config.DestinationConfig{
		{
			URL:     server1.URL,
			Method:  "POST",
			Headers: map[string]string{"Content-Type": "application/json", "X-Custom-Header": "custom-value"},
			Timeout: 5 * time.Second,
		},
		{
			URL:     server2.URL,
			Method:  "POST",
			Headers: map[string]string{"Content-Type": "application/json"},
			Timeout: 5 * time.Second,
		},
	}

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create proxy handler
	handler := NewProxyHandler(destinations, logger)

	// Forward webhook
	body := []byte(`{"event":"test"}`)
	headers := map[string]string{"User-Agent": "test-agent"}
	handler.ForwardWebhook(body, headers)

	// Add a small delay to allow goroutines to complete
	time.Sleep(100 * time.Millisecond)

	// Get metrics
	metrics := handler.GetMetrics()

	// Verify metrics
	assert.Equal(t, int64(2), metrics["total_requests"])
	assert.Equal(t, int64(1), metrics["successful_requests"])
	assert.Equal(t, int64(1), metrics["failed_requests"])
}

func TestMetrics(t *testing.T) {
	// Create metrics
	metrics := NewMetrics()

	// Record some metrics
	metrics.RecordRequest("https://example.com/webhook1")
	metrics.RecordSuccess("https://example.com/webhook1", 200, 100*time.Millisecond)
	metrics.RecordRequest("https://example.com/webhook1")
	metrics.RecordFailure("https://example.com/webhook1", "connection timeout", false)

	// Record a retry (which is a failure with retry=true)
	metrics.RecordFailure("https://example.com/webhook1", "connection timeout", true)

	metrics.RecordRequest("https://example.com/webhook2")
	metrics.RecordSuccess("https://example.com/webhook2", 201, 150*time.Millisecond)

	// Get metrics
	result := metrics.GetMetrics()

	// Verify global metrics
	assert.Equal(t, int64(3), result["total_requests"])
	assert.Equal(t, int64(2), result["successful_requests"])
	assert.Equal(t, int64(2), result["failed_requests"])
	assert.Equal(t, int64(1), result["retries"])

	// Verify destination metrics
	destinationsRaw, ok := result["destinations"]
	assert.True(t, ok, "destinations key should exist in metrics")

	destinations, ok := destinationsRaw.(map[string]interface{})
	assert.True(t, ok, "destinations should be a map[string]interface{}")

	webhook1Raw, ok := destinations["https://example.com/webhook1"]
	assert.True(t, ok, "webhook1 key should exist in destinations")

	webhook1, ok := webhook1Raw.(map[string]interface{})
	assert.True(t, ok, "webhook1 should be a map[string]interface{}")

	assert.Equal(t, int64(2), webhook1["total_requests"])
	assert.Equal(t, int64(1), webhook1["successful_requests"])
	assert.Equal(t, int64(2), webhook1["failed_requests"])
	assert.Equal(t, int64(1), webhook1["retries"])
	assert.Equal(t, "connection timeout", webhook1["last_error"])

	webhook2Raw, ok := destinations["https://example.com/webhook2"]
	assert.True(t, ok, "webhook2 key should exist in destinations")

	webhook2, ok := webhook2Raw.(map[string]interface{})
	assert.True(t, ok, "webhook2 should be a map[string]interface{}")

	assert.Equal(t, int64(1), webhook2["total_requests"])
	assert.Equal(t, int64(1), webhook2["successful_requests"])
	assert.Equal(t, int64(0), webhook2["failed_requests"])

	// Verify status codes
	statusCodesRaw, ok := result["status_codes"]
	assert.True(t, ok, "status_codes key should exist in metrics")

	// Use type assertion with ok check
	statusCodes, ok := statusCodesRaw.(map[int]int64)
	assert.True(t, ok, "status_codes should be a map[int]int64")

	assert.Equal(t, int64(1), statusCodes[200])
	assert.Equal(t, int64(1), statusCodes[201])

	// Reset metrics
	metrics.Reset()
}

// TestResetMetrics tests the ResetMetrics function
func TestResetMetrics(t *testing.T) {
	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create destinations
	destinations := []config.DestinationConfig{
		{
			URL:     "https://example.com",
			Method:  "POST",
			Headers: map[string]string{"Content-Type": "application/json"},
			Timeout: 5 * time.Second,
		},
	}

	// Create proxy handler
	handler := NewProxyHandler(destinations, logger)

	// Record some metrics
	handler.metrics.RecordRequest("https://example.com")
	handler.metrics.RecordSuccess("https://example.com", 200, 100*time.Millisecond)

	// Verify metrics before reset
	metrics := handler.GetMetrics()
	assert.Equal(t, int64(1), metrics["total_requests"])
	assert.Equal(t, int64(1), metrics["successful_requests"])

	// Reset metrics
	handler.ResetMetrics()

	// Verify metrics after reset
	metricsAfterReset := handler.GetMetrics()
	assert.Equal(t, int64(0), metricsAfterReset["total_requests"])
	assert.Equal(t, int64(0), metricsAfterReset["successful_requests"])
}

// TestShouldRetry tests the shouldRetry function
func TestShouldRetry(t *testing.T) {
	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create destinations
	destinations := []config.DestinationConfig{
		{
			URL:        "https://example.com",
			Method:     "POST",
			Headers:    map[string]string{"Content-Type": "application/json"},
			Timeout:    5 * time.Second,
			Retries:    3,
			RetryDelay: 100 * time.Millisecond,
		},
	}

	// Create proxy handler
	handler := NewProxyHandler(destinations, logger)

	// Test case 1: Should retry (attempt < maxAttempts)
	dest := destinations[0]
	result := handler.shouldRetry(1, 4, dest)
	assert.True(t, result, "Should retry when attempt < maxAttempts")

	// Test case 2: Should not retry (attempt >= maxAttempts)
	result = handler.shouldRetry(4, 4, dest)
	assert.False(t, result, "Should not retry when attempt >= maxAttempts")

	// Test case 3: Should retry with default retry delay (RetryDelay <= 0)
	destWithZeroDelay := config.DestinationConfig{
		URL:        "https://example.com",
		Method:     "POST",
		Headers:    map[string]string{"Content-Type": "application/json"},
		Timeout:    5 * time.Second,
		Retries:    3,
		RetryDelay: 0 * time.Millisecond,
	}
	result = handler.shouldRetry(1, 4, destWithZeroDelay)
	assert.True(t, result, "Should retry with default delay when RetryDelay <= 0")
}

// TestSendRequest tests the sendRequest function
func TestSendRequest(t *testing.T) {
	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Test case 1: Successful request
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-agent", r.Header.Get("User-Agent"))
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))

		// Return success
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok"}`))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server1.Close()

	dest1 := config.DestinationConfig{
		URL:     server1.URL,
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json", "X-Custom-Header": "custom-value"},
		Timeout: 5 * time.Second,
	}

	// Create proxy handler
	handler := NewProxyHandler([]config.DestinationConfig{dest1}, logger)

	// Send request
	client := &http.Client{Timeout: 5 * time.Second}
	body := []byte(`{"event":"test"}`)
	headers := map[string]string{"User-Agent": "test-agent"}
	statusCode, respBody, duration, err := handler.sendRequest(client, dest1, body, headers, false)

	// Verify response
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.Equal(t, `{"status":"ok"}`, string(respBody))
	assert.Greater(t, duration.Nanoseconds(), int64(0))

	// Test case 2: Failed request - server error
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Return error
		w.WriteHeader(http.StatusInternalServerError)
		_, writeErr := w.Write([]byte(`{"status":"error"}`))
		if writeErr != nil {
			t.Fatalf("Failed to write response: %v", writeErr)
		}
	}))
	defer server2.Close()

	dest2 := config.DestinationConfig{
		URL:     server2.URL,
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
		Timeout: 5 * time.Second,
	}

	// Send request
	statusCode, respBody, duration, err = handler.sendRequest(client, dest2, body, headers, false)

	// Verify response
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, statusCode)
	assert.Equal(t, `{"status":"error"}`, string(respBody))
	assert.Greater(t, duration.Nanoseconds(), int64(0))

	// Test case 3: Failed request - invalid URL
	destInvalid := config.DestinationConfig{
		URL:     "http://invalid-url-that-does-not-exist.example",
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
		Timeout: 500 * time.Millisecond,
	}

	// Send request
	statusCode, respBody, duration, err = handler.sendRequest(client, destInvalid, body, headers, true)

	// Verify response
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
	assert.Equal(t, 0, statusCode)
	assert.Nil(t, respBody)
	assert.Greater(t, duration.Nanoseconds(), int64(0))

	// Test case 4: Failed request - invalid request creation
	destInvalidMethod := config.DestinationConfig{
		URL:     "http://example.com",
		Method:  "\n", // Invalid method
		Headers: map[string]string{"Content-Type": "application/json"},
		Timeout: 5 * time.Second,
	}

	// Send request
	statusCode, respBody, _, err = handler.sendRequest(client, destInvalidMethod, body, headers, false)

	// Verify response
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
	assert.Equal(t, 0, statusCode)
	assert.Nil(t, respBody)
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

// TestSendRequestReadBodyError tests the sendRequest function when reading the response body fails
func TestSendRequestReadBodyError(t *testing.T) {
	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a test server that returns a response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok"}`))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Create a client with a transport that returns a response with a body that will fail to read
	client := &http.Client{
		Transport: &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       &MockReadCloser{},
				Request: &http.Request{
					Method: "POST",
					URL:    nil,
				},
			},
			err: nil,
		},
	}

	dest := config.DestinationConfig{
		URL:     server.URL,
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
		Timeout: 5 * time.Second,
	}

	// Create proxy handler
	handler := NewProxyHandler([]config.DestinationConfig{dest}, logger)

	// Send request
	body := []byte(`{"event":"test"}`)
	headers := map[string]string{"User-Agent": "test-agent"}
	statusCode, respBody, duration, err := handler.sendRequest(client, dest, body, headers, false)

	// Verify response
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read response body")
	assert.Equal(t, http.StatusOK, statusCode)
	assert.Nil(t, respBody)
	assert.Greater(t, duration.Nanoseconds(), int64(0))
}

// mockTransport is a mock http.RoundTripper
type mockTransport struct {
	response *http.Response
	err      error
}

func (m *mockTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return m.response, m.err
}

// TestForwardToDestination tests the forwardToDestination function
func TestForwardToDestination(t *testing.T) {
	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Test case 1: Successful forwarding
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Return success
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok"}`))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server1.Close()

	dest1 := config.DestinationConfig{
		URL:     server1.URL,
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
		Timeout: 5 * time.Second,
	}

	// Create proxy handler
	handler := NewProxyHandler([]config.DestinationConfig{dest1}, logger)

	// Forward webhook
	body := []byte(`{"event":"test"}`)
	headers := map[string]string{"User-Agent": "test-agent"}
	handler.forwardToDestination(dest1, body, headers)

	// Verify metrics
	metrics := handler.GetMetrics()
	assert.Equal(t, int64(1), metrics["total_requests"])
	assert.Equal(t, int64(1), metrics["successful_requests"])
	assert.Equal(t, int64(0), metrics["failed_requests"])

	// Test case 2: Failed forwarding with retries
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Return error
		w.WriteHeader(http.StatusInternalServerError)
		_, writeErr := w.Write([]byte(`{"status":"error"}`))
		if writeErr != nil {
			t.Fatalf("Failed to write response: %v", writeErr)
		}
	}))
	defer server2.Close()

	dest2 := config.DestinationConfig{
		URL:        server2.URL,
		Method:     "POST",
		Headers:    map[string]string{"Content-Type": "application/json"},
		Timeout:    5 * time.Second,
		Retries:    2,
		RetryDelay: 100 * time.Millisecond,
	}

	// Reset metrics
	handler.ResetMetrics()

	// Forward webhook
	handler.forwardToDestination(dest2, body, headers)

	// Verify metrics
	metrics = handler.GetMetrics()
	assert.Equal(t, int64(1), metrics["total_requests"])
	assert.Equal(t, int64(0), metrics["successful_requests"])
	assert.Equal(t, int64(3), metrics["failed_requests"]) // Initial attempt + 2 retries
	assert.Equal(t, int64(2), metrics["retries"])

	// Test case 3: Forwarding with negative retries (should default to 1 attempt)
	dest3 := config.DestinationConfig{
		URL:     server2.URL,
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
		Timeout: 5 * time.Second,
		Retries: -1,
	}

	// Reset metrics
	handler.ResetMetrics()

	// Forward webhook
	handler.forwardToDestination(dest3, body, headers)

	// Verify metrics
	metrics = handler.GetMetrics()
	assert.Equal(t, int64(1), metrics["total_requests"])
	assert.Equal(t, int64(0), metrics["successful_requests"])
	assert.Equal(t, int64(1), metrics["failed_requests"]) // Only 1 attempt
	assert.Equal(t, int64(0), metrics["retries"])
}

// TestForwardToDestinationWithRequestError tests the forwardToDestination function with a request error
func TestForwardToDestinationWithRequestError(t *testing.T) {
	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a destination with an invalid URL
	dest := config.DestinationConfig{
		URL:        "http://invalid-url-that-does-not-exist.example",
		Method:     "POST",
		Headers:    map[string]string{"Content-Type": "application/json"},
		Timeout:    500 * time.Millisecond,
		Retries:    1,
		RetryDelay: 100 * time.Millisecond,
	}

	// Create proxy handler
	handler := NewProxyHandler([]config.DestinationConfig{dest}, logger)

	// Forward webhook
	body := []byte(`{"event":"test"}`)
	headers := map[string]string{"User-Agent": "test-agent"}
	handler.forwardToDestination(dest, body, headers)

	// Verify metrics
	metrics := handler.GetMetrics()
	assert.Equal(t, int64(1), metrics["total_requests"])
	assert.Equal(t, int64(0), metrics["successful_requests"])
	assert.Equal(t, int64(2), metrics["failed_requests"]) // Initial attempt + 1 retry
	assert.Equal(t, int64(1), metrics["retries"])
}
