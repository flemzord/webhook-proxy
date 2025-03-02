package proxy

import (
	"sync"
	"time"
)

// Metrics represents the metrics for the proxy
type Metrics struct {
	mu                 sync.RWMutex
	totalRequests      int64
	successfulRequests int64
	failedRequests     int64
	retries            int64
	responseTimeTotal  time.Duration
	responseTimeCount  int64
	statusCodes        map[int]int64
	destinations       map[string]*DestinationMetrics
}

// DestinationMetrics represents metrics for a specific destination
type DestinationMetrics struct {
	totalRequests      int64
	successfulRequests int64
	failedRequests     int64
	retries            int64
	responseTimeTotal  time.Duration
	responseTimeCount  int64
	statusCodes        map[int]int64
	lastError          string
	lastErrorTime      time.Time
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		statusCodes:  make(map[int]int64),
		destinations: make(map[string]*DestinationMetrics),
	}
}

// RecordRequest records a request to a destination
func (m *Metrics) RecordRequest(destination string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++

	// Initialize destination metrics if not exists
	if _, exists := m.destinations[destination]; !exists {
		m.destinations[destination] = &DestinationMetrics{
			statusCodes: make(map[int]int64),
		}
	}

	m.destinations[destination].totalRequests++
}

// RecordSuccess records a successful request
func (m *Metrics) RecordSuccess(destination string, statusCode int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.successfulRequests++
	m.responseTimeTotal += duration
	m.responseTimeCount++
	m.statusCodes[statusCode]++

	// Update destination metrics
	if dest, exists := m.destinations[destination]; exists {
		dest.successfulRequests++
		dest.responseTimeTotal += duration
		dest.responseTimeCount++
		dest.statusCodes[statusCode]++
	}
}

// RecordFailure records a failed request
func (m *Metrics) RecordFailure(destination string, err string, retry bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failedRequests++
	if retry {
		m.retries++
	}

	// Update destination metrics
	if dest, exists := m.destinations[destination]; exists {
		dest.failedRequests++
		if retry {
			dest.retries++
		}
		dest.lastError = err
		dest.lastErrorTime = time.Now()
	}
}

// GetMetrics returns a copy of the current metrics
func (m *Metrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate average response time
	var avgResponseTime float64
	if m.responseTimeCount > 0 {
		avgResponseTime = float64(m.responseTimeTotal.Milliseconds()) / float64(m.responseTimeCount)
	}

	// Build destinations metrics
	destinations := make(map[string]interface{})
	for url, dest := range m.destinations {
		var destAvgResponseTime float64
		if dest.responseTimeCount > 0 {
			destAvgResponseTime = float64(dest.responseTimeTotal.Milliseconds()) / float64(dest.responseTimeCount)
		}

		destinations[url] = map[string]interface{}{
			"total_requests":       dest.totalRequests,
			"successful_requests":  dest.successfulRequests,
			"failed_requests":      dest.failedRequests,
			"retries":              dest.retries,
			"avg_response_time_ms": destAvgResponseTime,
			"status_codes":         dest.statusCodes,
			"last_error":           dest.lastError,
			"last_error_time":      dest.lastErrorTime,
		}
	}

	return map[string]interface{}{
		"total_requests":       m.totalRequests,
		"successful_requests":  m.successfulRequests,
		"failed_requests":      m.failedRequests,
		"retries":              m.retries,
		"avg_response_time_ms": avgResponseTime,
		"status_codes":         m.statusCodes,
		"destinations":         destinations,
	}
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests = 0
	m.successfulRequests = 0
	m.failedRequests = 0
	m.retries = 0
	m.responseTimeTotal = 0
	m.responseTimeCount = 0
	m.statusCodes = make(map[int]int64)
	m.destinations = make(map[string]*DestinationMetrics)
}
