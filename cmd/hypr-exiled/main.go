package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/rs/zerolog"

	"hypr-exiled/internal/app"
	"hypr-exiled/internal/ipc"
	"hypr-exiled/pkg/config"
	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/logger"
	"hypr-exiled/pkg/notify"
)

//go:embed assets/*
var embeddedAssets embed.FS

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "path to config file")
	debug := flag.Bool("debug", false, "enable debug logging")
	showTrades := flag.Bool("showTrades", false, "show the trades UI")
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

	log.Info("Starting Hypr Exiled",
		"version", "1.0.0",
		"pid", os.Getpid(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"debug", *debug)

	// Handle showTrades command
	if *showTrades {
		handleShowTrades(log, *configPath)
		return
	}

	// Otherwise, start the background service
	startBackgroundService(log, *configPath)
}

// handleShowTrades handles the --showTrades command.
func handleShowTrades(log *logger.Logger, configPath string) {
	log.Info("Showing trades UI")

	// Load minimal configuration for notifications
	log.Debug("Loading minimal configuration for notifications", "provided_path", configPath)
	config, err := config.FindConfig(configPath, log, embeddedAssets)
	if err != nil {
		log.Error("Failed to load configuration", err, "provided_path", configPath)
		// Use fmt to print the error since the notifier is not initialized yet
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		return
	}

	// Initialize global state (config, logger, and notifier)
	log.Debug("Initializing global notifier")
	global.InitGlobals(config, log, embeddedAssets)

	// Send the showTrades command to the background service
	resp, err := ipc.SendCommand("showTrades")
	if err != nil {
		log.Error("Failed to communicate with background service", err)
		global.GetNotifier().Show("Failed to communicate with background service. Is it running?", notify.Error)
		return
	}

	if resp.Status != "success" {
		log.Error("Failed to show trades", fmt.Errorf("message: %s", resp.Message))
		global.GetNotifier().Show(resp.Message, notify.Error)
		return
	}

	log.Info("Trades displayed successfully")
}

// startBackgroundService starts the background service.
func startBackgroundService(log *logger.Logger, configPath string) {
	// Load configuration
	log.Debug("Loading configuration", "provided_path", configPath)
	config, err := config.FindConfig(configPath, log, embeddedAssets)
	if err != nil {
		log.Error("Failed to load configuration", err,
			"provided_path", configPath)
		os.Exit(1)
	}
	log.Info("Configuration loaded successfully",
		"poe_log_path", config.GetPoeLogPath(),
		"trigger_count", len(config.GetTriggers()),
		"command_count", len(config.GetCommands()))

	// Initialize globals
	log.Debug("Initializing global instances")
	global.InitGlobals(config, log, embeddedAssets)
	log.Debug("Global instances initialized successfully")

	// Create and start the application
	log.Debug("Creating Hypr Exiled instance")
	app, err := app.NewHyprExiled()
	if err != nil {
		log.Fatal("Failed to create Hypr Exiled", err)
	}

	log.Info("Starting application")
	if err := app.Run(); err != nil {
		log.Fatal("Application error", err)
	}
}
