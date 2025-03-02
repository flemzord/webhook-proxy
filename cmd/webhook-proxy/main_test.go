package main

import (
	"flag"
	"io"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestVersionFlag tests the -version flag
func TestVersionFlag(t *testing.T) {
	// Save original os.Args and flag.CommandLine
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	// Reset flag.CommandLine to simulate a fresh run
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Set up test args
	os.Args = []string{"webhook-proxy", "-version"}

	// Set test version variables
	oldVersion := version
	oldCommit := commit
	oldDate := date
	version = "test-version"
	commit = "test-commit"
	date = "test-date"
	defer func() {
		version = oldVersion
		commit = oldCommit
		date = oldDate
	}()

	// Create a channel to capture exit code
	exitCh := make(chan int, 1)

	// Override exitFunc
	oldExit := exitFunc
	exitFunc = func(code int) {
		exitCh <- code
		// Don't actually exit in tests
	}
	defer func() {
		exitFunc = oldExit
	}()

	// Redirect all logrus output to discard to avoid "file already closed" errors
	oldOutput := logrus.StandardLogger().Out
	logrus.StandardLogger().SetOutput(io.Discard)
	defer logrus.StandardLogger().SetOutput(oldOutput)

	// Run main in a goroutine
	go func() {
		main()

		// If main returns without calling exitFunc, send 0
		select {
		case exitCh <- 0:
		default:
			// Channel already has a value, do nothing
		}
	}()

	// Get exit code
	exitCode := <-exitCh

	// Check exit code
	assert.Equal(t, 0, exitCode, "Expected exit code 0 when version flag is set")
}
