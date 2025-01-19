// main.go
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/rs/zerolog"

	"poe-helper/internal/app"
	"poe-helper/pkg/config"
	"poe-helper/pkg/global"
	"poe-helper/pkg/logger"
)

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

	// Initialize logger first for early logging
	log, err := logger.NewLogger(
		logger.WithConsole(),
		logger.WithLevel(logLevel),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	log.Info("Starting POE Helper",
		"version", "1.0.0",
		"pid", os.Getpid(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"debug", *debug)

	log.Debug("Initializing with parameters",
		"config_path", *configPath,
		"log_level", logLevel.String())

	// Load configuration
	log.Debug("Loading configuration", "provided_path", *configPath)
	config, err := config.FindConfig(*configPath, log)
	if err != nil {
		log.Error("Failed to load configuration", err,
			"provided_path", *configPath)
		os.Exit(1)
	}
	log.Info("Configuration loaded successfully",
		"poe_log_path", config.PoeLogPath,
		"trigger_count", len(config.Triggers),
		"command_count", len(config.Commands))

	// Initialize globals
	log.Debug("Initializing global instances")
	global.InitGlobals(config, log)
	log.Debug("Global instances initialized successfully")

	// Create and start the application
	log.Debug("Creating POE Helper instance")
	app, err := app.NewPOEHelper()
	if err != nil {
		log.Fatal("Failed to create POE Helper", err)
	}

	log.Info("Starting application")
	if err := app.Run(); err != nil {
		log.Fatal("Application error", err)
	}
}
