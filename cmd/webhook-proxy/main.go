package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/flemzord/webhook-proxy/config"
	"github.com/flemzord/webhook-proxy/logger"
	"github.com/flemzord/webhook-proxy/server"
	"github.com/sirupsen/logrus"
)

// Version information - will be injected at build time by GoReleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version information if requested
	if *showVersion {
		fmt.Printf("webhook-proxy version %s, commit %s, built at %s\n", version, commit, date)
		os.Exit(0)
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
		os.Exit(1)
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
		os.Exit(1)
	}
}
