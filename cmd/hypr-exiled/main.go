package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

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
	configPath := flag.String("config", "", "path to config file")
	debug := flag.Bool("debug", false, "enable debug logging")
	showTrades := flag.Bool("showTrades", false, "show the trades UI")
	hideout := flag.Bool("hideout", false, "go to hideout")
	flag.Parse()

	// Initialize logger
	logLevel := zerolog.InfoLevel
	if *debug {
		logLevel = zerolog.DebugLevel
	}

	log, err := logger.NewLogger(
		logger.WithConsole(),
		logger.WithLevel(logLevel),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger failed: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	// Route commands
	switch {
	case *showTrades:
		handleShowTrades(log, *configPath)
	case *hideout:
		handleHideout(log, *configPath)
	default:
		startBackgroundService(log, *configPath)
	}
}

// handleShowTrades handles the --showTrades command.
func handleShowTrades(log *logger.Logger, configPath string) {
	log.Info("Showing trades UI")
	_, cleanup, err := initializeCommon(log, configPath)
	if err != nil {
		log.Error("Initialization failed", err)
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}
	defer cleanup()

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
	cfg, cleanup, err := initializeCommon(log, configPath)
	if err != nil {
		log.Error("Initialization failed", err)
		os.Exit(1)
	}
	defer cleanup()

	// Create and start service
	log.Info("Service configuration loaded",
		"poe_log_path", cfg.GetPoeLogPath(),
		"triggers", len(cfg.GetTriggers()),
		"commands", len(cfg.GetCommands()))

	app, err := app.NewHyprExiled()
	if err != nil {
		log.Fatal("Failed to create Hypr Exiled", err)
	}

	log.Info("Starting application")
	if err := app.Run(); err != nil {
		log.Fatal("Application error", err)
	}
}

func handleHideout(log *logger.Logger, configPath string) {
	_, cleanup, err := initializeCommon(log, configPath)
	if err != nil {
		log.Error("Initialization failed", err)
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}
	defer cleanup()

	resp, err := ipc.SendCommand("hideout")
	if err != nil {
		log.Error("Hideout command failed", err)
		global.GetNotifier().Show("Failed to contact service", notify.Error)
		return
	}

	if resp.Status != "success" {
		log.Error("Hideout failed", fmt.Errorf(resp.Message))
		global.GetNotifier().Show(resp.Message, notify.Error)
		return
	}

	log.Info("Hideout command executed via IPC")
}

func initializeCommon(log *logger.Logger, configPath string) (*config.Config, func(), error) {
	// Load configuration
	log.Debug("Loading configuration", "path", configPath)
	cfg, err := config.FindConfig(configPath, log, embeddedAssets)
	if err != nil {
		return nil, nil, fmt.Errorf("config error: %w", err)
	}

	// Initialize global state
	log.Debug("Initializing global instances")
	global.InitGlobals(cfg, log, embeddedAssets)

	// Return cleanup function to close resources
	cleanup := func() {
		global.Close()
	}

	return cfg, cleanup, nil
}
