package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"poe-helper/internal/app"
	"poe-helper/pkg/logger"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "path to config file")
	logPath := flag.String("log", "", "path to log file")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	// Setup logging
	logLevel := zerolog.InfoLevel
	if *debug {
		logLevel = zerolog.DebugLevel
	}

	// Determine log file path
	if *logPath == "" {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get config directory: %v\n", err)
			os.Exit(1)
		}
		*logPath = filepath.Join(userConfigDir, "poe-helper", "poe-helper.log")
	}

	// Initialize logger
	log, err := logger.NewLogger(
		logger.WithFile(*logPath),
		logger.WithConsole(),
		logger.WithLevel(logLevel),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	// Create and run application
	config := &app.Config{}
	if *configPath != "" {
		if err := config.LoadFromFile(*configPath); err != nil {
			log.Fatal("Failed to load config", err)
		}
	}

	app, err := app.NewPOEHelper(config, log, *debug)
	if err != nil {
		log.Fatal("Failed to create POE Helper", err)
	}

	if err := app.Run(); err != nil {
		log.Fatal("Application error", err)
	}
}
