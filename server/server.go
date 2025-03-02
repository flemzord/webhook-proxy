package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/flemzord/webhook-proxy/config"
	"github.com/flemzord/webhook-proxy/logger"
	"github.com/flemzord/webhook-proxy/proxy"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

// Server represents the HTTP server
type Server struct {
	config *config.Config
	router *chi.Mux
	log    *logrus.Logger
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
		config: cfg,
		router: router,
		log:    log,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Register routes for each endpoint
	for _, endpoint := range s.config.Endpoints {
		s.registerEndpoint(endpoint)
	}

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

// readRequestBody reads the request body
func readRequestBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()

	// Read the body
	body := make([]byte, r.ContentLength)
	_, err := r.Body.Read(body)
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}

	return body, nil
}
