package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/flemzord/webhook-proxy/config"
	"github.com/flemzord/webhook-proxy/logger"
	"github.com/sirupsen/logrus"
)

// Handler handles forwarding webhooks to destinations
type Handler struct {
	destinations []config.DestinationConfig
	client       *http.Client
	log          *logrus.Logger
	metrics      *Metrics
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(destinations []config.DestinationConfig, log *logrus.Logger) *Handler {
	// Create HTTP client with reasonable defaults
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return &Handler{
		destinations: destinations,
		client:       client,
		log:          log,
		metrics:      NewMetrics(),
	}
}

// ForwardWebhook forwards a webhook to all configured destinations
func (p *Handler) ForwardWebhook(body []byte, headers map[string]string) {
	var wg sync.WaitGroup

	for _, dest := range p.destinations {
		wg.Add(1)
		// Forward to each destination in a separate goroutine
		go func(d config.DestinationConfig) {
			defer wg.Done()
			p.forwardToDestination(d, body, headers)
		}(dest)
	}

	// Wait for all forwarding operations to complete (optional)
	// If you want to return immediately to the caller, comment this out
	// wg.Wait()
}

// GetMetrics returns the current metrics
func (p *Handler) GetMetrics() map[string]interface{} {
	return p.metrics.GetMetrics()
}

// ResetMetrics resets all metrics
func (p *Handler) ResetMetrics() {
	p.metrics.Reset()
}

// forwardToDestination forwards a webhook to a single destination
func (p *Handler) forwardToDestination(dest config.DestinationConfig, body []byte, headers map[string]string) {
	// Record the request in metrics
	p.metrics.RecordRequest(dest.URL)

	// Set client timeout for this specific request
	client := &http.Client{
		Timeout: dest.Timeout,
	}

	// Retry logic
	maxAttempts := dest.Retries + 1 // +1 for the initial attempt
	if maxAttempts <= 0 {
		maxAttempts = 1 // At least one attempt
	}

	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		isRetry := attempt > 1

		// Send the request
		statusCode, respBody, duration, err := p.sendRequest(client, dest, body, headers, isRetry)
		if err != nil {
			lastErr = err

			// If this is not the last attempt, wait before retrying
			if p.shouldRetry(attempt, maxAttempts, dest) {
				continue
			}
			return
		}

		// If successful (2xx status code), log and return
		if statusCode >= 200 && statusCode < 300 {
			// Record success in metrics
			p.metrics.RecordSuccess(dest.URL, statusCode, duration)

			// Log success with more details
			p.log.WithFields(logrus.Fields{
				"destination":   dest.URL,
				"status_code":   statusCode,
				"duration_ms":   duration.Milliseconds(),
				"attempt":       attempt,
				"response_size": len(respBody),
			}).Info("Webhook forwarded successfully")

			return
		}

		// If we got a non-2xx status code and have retries left
		lastErr = fmt.Errorf("received non-2xx status code: %d, body: %s", statusCode, string(respBody))
		logger.LogWebhookError(p.log, dest.URL, lastErr, attempt, maxAttempts)

		// Record failure in metrics
		p.metrics.RecordFailure(dest.URL, lastErr.Error(), isRetry)

		if p.shouldRetry(attempt, maxAttempts, dest) {
			// Log retry attempt with more details
			p.log.WithFields(logrus.Fields{
				"destination":   dest.URL,
				"status_code":   statusCode,
				"attempt":       attempt,
				"max_attempts":  maxAttempts,
				"retry_delay":   dest.RetryDelay,
				"response_body": string(respBody),
			}).Info("Retrying webhook forwarding due to non-2xx status code")
		}
	}

	// If we've exhausted all retries, log a final error
	if lastErr != nil {
		p.log.WithFields(logrus.Fields{
			"destination": dest.URL,
			"error":       lastErr,
			"attempts":    maxAttempts,
		}).Error("Webhook forwarding failed after all retry attempts")
	}
}

// sendRequest sends a request to the destination and returns the status code, response body, duration, and error
func (p *Handler) sendRequest(client *http.Client, dest config.DestinationConfig, body []byte, headers map[string]string, isRetry bool) (int, []byte, time.Duration, error) {
	// Create request with context for better timeout handling
	ctx, cancel := context.WithTimeout(context.Background(), dest.Timeout)
	defer cancel() // Cancel the context to prevent resource leaks

	req, err := http.NewRequestWithContext(ctx, dest.Method, dest.URL, bytes.NewReader(body))
	if err != nil {
		lastErr := fmt.Errorf("failed to create request: %w", err)
		p.log.WithFields(logrus.Fields{
			"error":       err,
			"destination": dest.URL,
			"method":      dest.Method,
		}).Error("Failed to create request")

		// Record failure in metrics
		p.metrics.RecordFailure(dest.URL, lastErr.Error(), isRetry)
		return 0, nil, 0, lastErr
	}

	// Add headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Add custom headers from configuration
	for k, v := range dest.Headers {
		req.Header.Set(k, v)
	}

	// Send request and measure time
	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		lastErr := fmt.Errorf("request failed: %w", err)
		logger.LogWebhookError(p.log, dest.URL, err, 1, 1)

		// Record failure in metrics
		p.metrics.RecordFailure(dest.URL, lastErr.Error(), isRetry)
		return 0, nil, duration, lastErr
	}

	// Get status code
	statusCode := resp.StatusCode

	// Read and close response body
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		lastErr := fmt.Errorf("failed to read response body: %w", err)
		logger.LogWebhookError(p.log, dest.URL, err, 1, 1)

		// Record failure in metrics
		p.metrics.RecordFailure(dest.URL, lastErr.Error(), isRetry)
		return statusCode, nil, duration, lastErr
	}

	return statusCode, respBody, duration, nil
}

// shouldRetry determines if a retry should be attempted
func (p *Handler) shouldRetry(attempt, maxAttempts int, dest config.DestinationConfig) bool {
	if attempt >= maxAttempts {
		return false
	}

	retryDelay := dest.RetryDelay
	if retryDelay <= 0 {
		retryDelay = 1 * time.Second
	}

	// Log retry attempt
	p.log.WithFields(logrus.Fields{
		"destination":  dest.URL,
		"attempt":      attempt,
		"max_attempts": maxAttempts,
		"retry_delay":  retryDelay,
	}).Info("Retrying webhook forwarding")

	time.Sleep(retryDelay)
	return true
}
