package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flemzord/webhook-proxy/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestProxyHandler_ForwardWebhook(t *testing.T) {
	// Create test servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-value", r.Header.Get("X-Test-Header"))

		// Return success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))

		// Return error to test retry
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status":"error"}`))
	}))
	defer server2.Close()

	// Create destinations
	destinations := []config.DestinationConfig{
		{
			URL:        server1.URL,
			Method:     "POST",
			Timeout:    5 * time.Second,
			Retries:    0,
			RetryDelay: 1 * time.Second,
			Headers:    map[string]string{},
		},
		{
			URL:        server2.URL,
			Method:     "POST",
			Timeout:    5 * time.Second,
			Retries:    2,
			RetryDelay: 100 * time.Millisecond, // Short delay for tests
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
			},
		},
	}

	// Create logger
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	// Create proxy handler
	handler := NewProxyHandler(destinations, log)

	// Create test webhook
	body := []byte(`{"event":"test"}`)
	headers := map[string]string{
		"Content-Type":  "application/json",
		"X-Test-Header": "test-value",
	}

	// Forward webhook
	handler.ForwardWebhook(body, headers)

	// Wait for goroutines to complete
	time.Sleep(500 * time.Millisecond)

	// Check metrics
	metrics := handler.GetMetrics()

	// Verify metrics
	assert.Equal(t, int64(2), metrics["total_requests"])

	// Reset metrics
	handler.ResetMetrics()

	// Verify metrics were reset
	metrics = handler.GetMetrics()
	assert.Equal(t, int64(0), metrics["total_requests"])
}

func TestMetrics(t *testing.T) {
	// Create metrics
	metrics := NewMetrics()

	// Record requests
	metrics.RecordRequest("https://example.com/webhook1")
	metrics.RecordRequest("https://example.com/webhook2")
	metrics.RecordRequest("https://example.com/webhook1")

	// Record success
	metrics.RecordSuccess("https://example.com/webhook1", 200, 150*time.Millisecond)
	metrics.RecordSuccess("https://example.com/webhook2", 201, 200*time.Millisecond)

	// Record failure
	metrics.RecordFailure("https://example.com/webhook1", "connection timeout", true)

	// Get metrics
	result := metrics.GetMetrics()

	// Verify global metrics
	assert.Equal(t, int64(3), result["total_requests"])
	assert.Equal(t, int64(2), result["successful_requests"])
	assert.Equal(t, int64(1), result["failed_requests"])
	assert.Equal(t, int64(1), result["retries"])

	// Verify destination metrics
	destinations := result["destinations"].(map[string]interface{})

	webhook1 := destinations["https://example.com/webhook1"].(map[string]interface{})
	assert.Equal(t, int64(2), webhook1["total_requests"])
	assert.Equal(t, int64(1), webhook1["successful_requests"])
	assert.Equal(t, int64(1), webhook1["failed_requests"])
	assert.Equal(t, int64(1), webhook1["retries"])
	assert.Equal(t, "connection timeout", webhook1["last_error"])

	webhook2 := destinations["https://example.com/webhook2"].(map[string]interface{})
	assert.Equal(t, int64(1), webhook2["total_requests"])
	assert.Equal(t, int64(1), webhook2["successful_requests"])
	assert.Equal(t, int64(0), webhook2["failed_requests"])

	// Verify status codes
	statusCodes := result["status_codes"].(map[int]int64)
	assert.Equal(t, int64(1), statusCodes[200])
	assert.Equal(t, int64(1), statusCodes[201])

	// Reset metrics
	metrics.Reset()

	// Verify reset
	result = metrics.GetMetrics()
	assert.Equal(t, int64(0), result["total_requests"])
}
