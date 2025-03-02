package proxy

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/flemzord/webhook-proxy/config"
	"github.com/flemzord/webhook-proxy/logger"
	"github.com/sirupsen/logrus"
)

// ProxyHandler handles forwarding webhooks to destinations
type ProxyHandler struct {
	destinations []config.DestinationConfig
	client       *http.Client
	log          *logrus.Logger
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(destinations []config.DestinationConfig, log *logrus.Logger) *ProxyHandler {
	// Create HTTP client with reasonable defaults
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return &ProxyHandler{
		destinations: destinations,
		client:       client,
		log:          log,
	}
}

// ForwardWebhook forwards a webhook to all configured destinations
func (p *ProxyHandler) ForwardWebhook(body []byte, headers map[string]string) {
	for _, dest := range p.destinations {
		// Forward to each destination in a separate goroutine
		go p.forwardToDestination(dest, body, headers)
	}
}

// forwardToDestination forwards a webhook to a single destination
func (p *ProxyHandler) forwardToDestination(dest config.DestinationConfig, body []byte, headers map[string]string) {
	// Set client timeout for this specific request
	client := &http.Client{
		Timeout: dest.Timeout,
	}

	// Retry logic
	maxAttempts := dest.Retries
	if maxAttempts <= 0 {
		maxAttempts = 1 // At least one attempt
	}

	var statusCode int
	var duration float64

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Create request
		req, err := http.NewRequest(dest.Method, dest.URL, bytes.NewReader(body))
		if err != nil {
			p.log.WithFields(logrus.Fields{
				"error":       err,
				"destination": dest.URL,
				"method":      dest.Method,
			}).Error("Failed to create request")
			return
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
		duration = float64(time.Since(startTime).Milliseconds())

		if err != nil {
			logger.LogWebhookError(p.log, dest.URL, err, attempt, maxAttempts)

			// If this is not the last attempt, wait before retrying
			if attempt < maxAttempts {
				retryDelay := dest.RetryDelay
				if retryDelay <= 0 {
					retryDelay = 1 * time.Second
				}
				time.Sleep(retryDelay)
				continue
			}
			return
		}

		// Get status code and close response body
		statusCode = resp.StatusCode
		resp.Body.Close()

		// If successful (2xx status code), log and return
		if statusCode >= 200 && statusCode < 300 {
			logger.LogWebhookForwarded(p.log, dest.URL, statusCode, duration)
			return
		}

		// If we got a non-2xx status code and have retries left
		logger.LogWebhookError(p.log, dest.URL,
			fmt.Errorf("received non-2xx status code: %d", statusCode),
			attempt, maxAttempts)

		if attempt < maxAttempts {
			retryDelay := dest.RetryDelay
			if retryDelay <= 0 {
				retryDelay = 1 * time.Second
			}
			time.Sleep(retryDelay)
		}
	}
}
