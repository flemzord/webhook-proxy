package proxy

import (
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
