package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/rs/zerolog"

	"hypr-exiled/internal/app"
	"hypr-exiled/pkg/config"
	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/logger"
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

	log.Debug("Initializing with parameters",
		"config_path", *configPath,
		"log_level", logLevel.String())

	// Load configuration
	log.Debug("Loading configuration", "provided_path", *configPath)

	config, err := config.FindConfig(*configPath, log, embeddedAssets)
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

	// Handle showTrades command
	if *showTrades {
		log.Info("Showing trades UI")
		app, err := app.NewHyprExiled()
		if err != nil {
			log.Fatal("Failed to create Hypr Exiled", err)
		}
		if err := app.TradeManager.ShowTrades(); err != nil {
			log.Fatal("Failed to show trades", err)
		}
		return
	}

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
