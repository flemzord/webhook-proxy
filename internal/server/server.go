package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/flemzord/webhook-proxy/internal/config"
	"github.com/flemzord/webhook-proxy/internal/logger"
	"github.com/flemzord/webhook-proxy/internal/proxy"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

// Server represents the HTTP server
type Server struct {
	config        *config.Config
	router        *chi.Mux
	log           *logrus.Logger
	proxyHandlers map[string]*proxy.Handler
	version       string
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, log *logrus.Logger) *Server {
	router := chi.NewRouter()

	// Add middleware
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(30 * time.Second))

	// Add custom logger middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Process request
			next.ServeHTTP(w, r)

			// Log after request
			logger.LogWebhookReceived(
				log,
				r.URL.Path,
				r.Method,
				r.RemoteAddr,
				r.ContentLength,
			)
		})
	})

	return &Server{
		config:        cfg,
		router:        router,
		log:           log,
		proxyHandlers: make(map[string]*proxy.Handler),
		version:       "1.0.0", // This will be updated from main
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Register routes for each endpoint
	for _, endpoint := range s.config.Endpoints {
		s.registerEndpoint(endpoint)
	}

	// Register metrics endpoint
	s.registerMetricsEndpoint()

	// Register health check endpoint
	s.registerHealthCheckEndpoint()

	// Start server
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.log.WithFields(logrus.Fields{
		"address": addr,
	}).Info("Starting HTTP server")

	return http.ListenAndServe(addr, s.router)
}

// registerEndpoint registers a webhook endpoint
func (s *Server) registerEndpoint(endpoint config.EndpointConfig) {
	s.log.WithFields(logrus.Fields{
		"path":         endpoint.Path,
		"destinations": len(endpoint.Destinations),
	}).Info("Registering webhook endpoint")

	// Create a proxy handler for this endpoint
	proxyHandler := proxy.NewProxyHandler(endpoint.Destinations, s.log)

	// Store the proxy handler for metrics access
	s.proxyHandlers[endpoint.Path] = proxyHandler

	// Register the endpoint
	s.router.Post(endpoint.Path, func(w http.ResponseWriter, r *http.Request) {
		// Read the request body
		var body []byte
		var err error

		// Limit the body size to 10MB
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		body, err = readRequestBody(r)
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"error": err,
				"path":  endpoint.Path,
			}).Error("Failed to read request body")
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}

		// Get the headers
		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		// Forward the webhook
		go proxyHandler.ForwardWebhook(body, headers)

		// Return success immediately
		w.WriteHeader(http.StatusOK)
	})
}

// registerMetricsEndpoint registers the metrics endpoint
func (s *Server) registerMetricsEndpoint() {
	s.router.Get("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		// Collect metrics from all proxy handlers
		metrics := make(map[string]interface{})

		// Add global metrics
		var totalRequests int64
		var successfulRequests int64
		var failedRequests int64
		var retries int64

		// Collect metrics from each proxy handler
		endpointMetrics := make(map[string]interface{})
		for path, handler := range s.proxyHandlers {
			handlerMetrics := handler.GetMetrics()
			endpointMetrics[path] = handlerMetrics

			// Aggregate global metrics
			if val, ok := handlerMetrics["total_requests"].(int64); ok {
				totalRequests += val
			}
			if val, ok := handlerMetrics["successful_requests"].(int64); ok {
				successfulRequests += val
			}
			if val, ok := handlerMetrics["failed_requests"].(int64); ok {
				failedRequests += val
			}
			if val, ok := handlerMetrics["retries"].(int64); ok {
				retries += val
			}
		}

		// Build the complete metrics response
		metrics["global"] = map[string]interface{}{
			"total_requests":      totalRequests,
			"successful_requests": successfulRequests,
			"failed_requests":     failedRequests,
			"retries":             retries,
			"success_rate":        calculateSuccessRate(successfulRequests, totalRequests),
		}
		metrics["endpoints"] = endpointMetrics
		metrics["timestamp"] = time.Now().Format(time.RFC3339)

		// Return metrics as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metrics); err != nil {
			s.log.WithError(err).Error("Failed to encode metrics response")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	// Add endpoint to reset metrics
	s.router.Post("/metrics/reset", func(w http.ResponseWriter, _ *http.Request) {
		// Reset metrics for all proxy handlers
		for _, handler := range s.proxyHandlers {
			handler.ResetMetrics()
		}

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok","message":"Metrics reset successfully"}`))
		if err != nil {
			s.log.WithError(err).Error("Failed to write response")
		}
	})
}

// registerHealthCheckEndpoint registers the health check endpoint
func (s *Server) registerHealthCheckEndpoint() {
	s.router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		health := map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   s.version,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(health); err != nil {
			s.log.WithError(err).Error("Failed to encode health response")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})
}

// calculateSuccessRate calculates the success rate as a percentage
func calculateSuccessRate(successful, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(successful) / float64(total) * 100
}

// readRequestBody reads the request body
func readRequestBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()

	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// SetVersion sets the version of the server
func (s *Server) SetVersion(version string) {
	s.version = version
}
