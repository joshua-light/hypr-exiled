package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"poe-helper/internal/app"
	"poe-helper/pkg/logger"
)

// TODO: Add inf logs about config being loaded and from where
func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "path to config file")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	// Setup logging level
	logLevel := zerolog.InfoLevel
	if *debug {
		logLevel = zerolog.DebugLevel
	}

	// Load configuration with priority order
	config, err := app.FindConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get config directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger with just console and level options
	// It will use the default log path automatically
	log, err := logger.NewLogger(
		logger.WithConsole(),
		logger.WithLevel(logLevel),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	app, err := app.NewPOEHelper(config, log)
	if err != nil {
		log.Fatal("Failed to create POE Helper", err)
	}

	if err := app.Run(); err != nil {
		log.Fatal("Application error", err)
	}
}
