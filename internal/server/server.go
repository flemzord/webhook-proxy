package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/flemzord/webhook-proxy/internal/config"
	"github.com/flemzord/webhook-proxy/internal/logger"
	"github.com/flemzord/webhook-proxy/internal/proxy"
	"github.com/flemzord/webhook-proxy/internal/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/codes"
)

// Server represents the HTTP server
type Server struct {
	config        *config.Config
	router        *chi.Mux
	log           *logrus.Logger
	proxyHandlers map[string]*proxy.Handler
	version       string
	tracer        *telemetry.Tracer
}

// HTTPServerFunc is a function type that matches http.ListenAndServe
type HTTPServerFunc func(addr string, handler http.Handler) error

// DefaultHTTPServerFunc is the default implementation using http.ListenAndServe
var DefaultHTTPServerFunc HTTPServerFunc = http.ListenAndServe

// TracerShutdowner is an interface for shutting down a tracer
type TracerShutdowner interface {
	Shutdown(ctx context.Context) error
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, log *logrus.Logger) *Server {
	router := chi.NewRouter()

	// Add middleware
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(30 * time.Second))

	// Create a tracer
	tracer, err := telemetry.NewTracer(context.Background(), telemetry.Config{
		ServiceName:    "webhook-proxy",
		ServiceVersion: "1.0.0", // This will be updated with SetVersion
		ExporterType:   cfg.Telemetry.ExporterType,
		Endpoint:       cfg.Telemetry.Endpoint,
		Enabled:        cfg.Telemetry.Enabled,
	}, log)
	if err != nil {
		log.WithError(err).Warn("Failed to create tracer, using noop tracer")
		tracer = telemetry.NewNoopTracer()
	}

	server := &Server{
		config:        cfg,
		router:        router,
		log:           log,
		proxyHandlers: make(map[string]*proxy.Handler),
		version:       "1.0.0",
		tracer:        tracer,
	}

	// Add custom logger and tracing middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a span for the request
			ctx, span := server.tracer.StartSpan(r.Context(), "http.request")
			defer span.End()

			// Add request attributes to the span
			telemetry.AddAttribute(ctx, "http.method", r.Method)
			telemetry.AddAttribute(ctx, "http.url", r.URL.String())
			telemetry.AddAttribute(ctx, "http.host", r.Host)
			telemetry.AddAttribute(ctx, "http.user_agent", r.UserAgent())
			telemetry.AddAttribute(ctx, "http.request_id", middleware.GetReqID(ctx))

			// Update the request with the new context
			r = r.WithContext(ctx)

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

	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.StartWithServerFunc(DefaultHTTPServerFunc)
}

// StartWithServerFunc starts the HTTP server using the provided server function
func (s *Server) StartWithServerFunc(serverFunc HTTPServerFunc) error {
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

	return serverFunc(addr, s.router)
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
		// Get the parent span from the context
		ctx := r.Context()

		// Create a span for handling the webhook
		ctx, span := s.tracer.StartSpan(ctx, "webhook.handle")
		defer span.End()

		// Add endpoint attributes to the span
		telemetry.AddAttribute(ctx, "webhook.path", endpoint.Path)
		telemetry.AddAttribute(ctx, "webhook.destinations", len(endpoint.Destinations))

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

			// Record the error in the span
			telemetry.RecordError(ctx, err)
			telemetry.SetStatus(ctx, codes.Error, "Failed to read request body")

			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}

		// Add body size to the span
		telemetry.AddAttribute(ctx, "webhook.body_size", len(body))

		// Get the headers
		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		// Forward the webhook in a goroutine with the trace context
		go func() {
			// Create a new context for the goroutine
			forwardCtx, forwardSpan := s.tracer.StartSpan(context.Background(), "webhook.forward")
			defer forwardSpan.End()

			// Add attributes to the forward span
			telemetry.AddAttribute(forwardCtx, "webhook.path", endpoint.Path)
			telemetry.AddAttribute(forwardCtx, "webhook.destinations", len(endpoint.Destinations))
			telemetry.AddAttribute(forwardCtx, "webhook.body_size", len(body))

			// Forward the webhook
			proxyHandler.ForwardWebhook(body, headers)

			// Set success status
			telemetry.SetStatus(forwardCtx, codes.Ok, "Webhook forwarded")
		}()

		// Return a success response
		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"status":"accepted"}`))
		if err != nil {
			s.log.WithError(err).Error("Failed to write response")
		}

		// Set success status for the main span
		telemetry.SetStatus(ctx, codes.Ok, "Webhook accepted")
	})
}

// registerMetricsEndpoint registers the metrics endpoint
func (s *Server) registerMetricsEndpoint() {
	s.router.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Get the parent span from the context
		ctx := r.Context()

		// Create a span for handling the metrics request
		ctx, span := s.tracer.StartSpan(ctx, "metrics.get")
		defer span.End()

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

		// Add metrics to the span
		telemetry.AddAttribute(ctx, "metrics.total_requests", totalRequests)
		telemetry.AddAttribute(ctx, "metrics.successful_requests", successfulRequests)
		telemetry.AddAttribute(ctx, "metrics.failed_requests", failedRequests)
		telemetry.AddAttribute(ctx, "metrics.retries", retries)
		telemetry.AddAttribute(ctx, "metrics.success_rate", calculateSuccessRate(successfulRequests, totalRequests))
		telemetry.AddAttribute(ctx, "metrics.endpoint_count", len(endpointMetrics))

		// Return metrics as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metrics); err != nil {
			s.log.WithError(err).Error("Failed to encode metrics response")

			// Record the error in the span
			telemetry.RecordError(ctx, err)
			telemetry.SetStatus(ctx, codes.Error, "Failed to encode metrics response")

			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Set success status
		telemetry.SetStatus(ctx, codes.Ok, "Metrics returned successfully")
	})

	// Add endpoint to reset metrics
	s.router.Post("/metrics/reset", func(w http.ResponseWriter, r *http.Request) {
		// Get the parent span from the context
		ctx := r.Context()

		// Create a span for handling the metrics reset request
		ctx, span := s.tracer.StartSpan(ctx, "metrics.reset")
		defer span.End()

		// Reset metrics for all proxy handlers
		for _, handler := range s.proxyHandlers {
			handler.ResetMetrics()
		}

		// Add reset info to the span
		telemetry.AddAttribute(ctx, "metrics.reset", true)
		telemetry.AddAttribute(ctx, "metrics.endpoint_count", len(s.proxyHandlers))

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok","message":"Metrics reset successfully"}`))
		if err != nil {
			s.log.WithError(err).Error("Failed to write response")

			// Record the error in the span
			telemetry.RecordError(ctx, err)
			telemetry.SetStatus(ctx, codes.Error, "Failed to write response")
			return
		}

		// Set success status
		telemetry.SetStatus(ctx, codes.Ok, "Metrics reset successfully")
	})
}

// registerHealthCheckEndpoint registers the health check endpoint
func (s *Server) registerHealthCheckEndpoint() {
	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		// Get the parent span from the context
		ctx := r.Context()

		// Create a span for handling the health check request
		ctx, span := s.tracer.StartSpan(ctx, "health.check")
		defer span.End()

		health := map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   s.version,
		}

		// Add health info to the span
		telemetry.AddAttribute(ctx, "health.status", "ok")
		telemetry.AddAttribute(ctx, "health.version", s.version)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(health); err != nil {
			s.log.WithError(err).Error("Failed to encode health response")

			// Record the error in the span
			telemetry.RecordError(ctx, err)
			telemetry.SetStatus(ctx, codes.Error, "Failed to encode health response")

			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Set success status
		telemetry.SetStatus(ctx, codes.Ok, "Health check successful")
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

	// Update the tracer's service version if it's enabled
	if s.tracer != nil {
		s.updateTracer(version)
	}
}

// updateTracer creates a new tracer with the updated version
func (s *Server) updateTracer(version string) {
	// Create a new tracer with the updated version
	newTracer, err := telemetry.NewTracer(context.Background(), telemetry.Config{
		ServiceName:    "webhook-proxy", // Use the default service name
		ServiceVersion: version,
		ExporterType:   s.config.Telemetry.ExporterType,
		Endpoint:       s.config.Telemetry.Endpoint,
		Enabled:        s.config.Telemetry.Enabled,
	}, s.log)

	if err != nil {
		s.log.WithError(err).Warn("Failed to update tracer version")
		return
	}

	// Shutdown the old tracer only if we successfully created a new one
	if newTracer != nil {
		if err := s.tracer.Shutdown(context.Background()); err != nil {
			s.log.WithError(err).Warn("Failed to shutdown old tracer")
		}

		// Set the new tracer
		s.tracer = newTracer
	}
}
