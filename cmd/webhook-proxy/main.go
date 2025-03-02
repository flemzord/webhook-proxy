package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/flemzord/webhook-proxy/internal/config"
	"github.com/flemzord/webhook-proxy/internal/logger"
	"github.com/flemzord/webhook-proxy/internal/server"
	"github.com/sirupsen/logrus"
)

// Version information - will be injected at build time by GoReleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// exitFunc allows us to override the exit behavior for testing
var exitFunc = os.Exit

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version information if requested
	if *showVersion {
		fmt.Printf("webhook-proxy version %s, commit %s, built at %s\n", version, commit, date)
		exitFunc(0)
	}

	// Initialize logger
	log := logger.NewLogger()
	log.WithFields(logrus.Fields{
		"version": version,
		"commit":  commit,
		"built":   date,
	}).Info("Starting webhook-proxy")

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
			"path":  *configPath,
		}).Fatal("Failed to load configuration")
		exitFunc(1)
	}

	// Configure logger based on config
	logger.ConfigureLogger(log, cfg.Logging)

	// Initialize and start HTTP server
	srv := server.NewServer(cfg, log)
	srv.SetVersion(version)
	if err := srv.Start(); err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed to start server")
		exitFunc(1)
	}
}
