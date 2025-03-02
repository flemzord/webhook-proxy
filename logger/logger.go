package logger

import (
	"io"
	"os"

	"github.com/flemzord/webhook-proxy/config"
	"github.com/sirupsen/logrus"
)

// NewLogger creates a new logger instance
func NewLogger() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)
	return log
}

// ConfigureLogger configures the logger based on the configuration
func ConfigureLogger(log *logrus.Logger, cfg config.LoggingConfig) {
	// Configure log level
	switch cfg.Level {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}

	// Configure log format
	switch cfg.Format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	default:
		log.SetFormatter(&logrus.JSONFormatter{})
	}

	// Configure log output
	switch cfg.Output {
	case "stdout":
		log.SetOutput(os.Stdout)
	case "file":
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
				"path":  cfg.FilePath,
			}).Error("Failed to open log file, using stdout instead")
			log.SetOutput(os.Stdout)
		} else {
			log.SetOutput(io.MultiWriter(os.Stdout, file))
		}
	default:
		log.SetOutput(os.Stdout)
	}
}

// LogWebhookReceived logs information about a received webhook
func LogWebhookReceived(log *logrus.Logger, path string, method string, remoteAddr string, contentLength int64) {
	log.WithFields(logrus.Fields{
		"path":           path,
		"method":         method,
		"remote_addr":    remoteAddr,
		"content_length": contentLength,
	}).Info("Webhook received")
}

// LogWebhookForwarded logs information about a forwarded webhook
func LogWebhookForwarded(log *logrus.Logger, destination string, statusCode int, duration float64) {
	log.WithFields(logrus.Fields{
		"destination": destination,
		"status_code": statusCode,
		"duration_ms": duration,
	}).Info("Webhook forwarded")
}

// LogWebhookError logs information about a webhook forwarding error
func LogWebhookError(log *logrus.Logger, destination string, err error, attempt int, maxAttempts int) {
	log.WithFields(logrus.Fields{
		"destination":  destination,
		"error":        err,
		"attempt":      attempt,
		"max_attempts": maxAttempts,
	}).Error("Webhook forwarding failed")
}
