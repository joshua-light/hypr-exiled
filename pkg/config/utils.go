package config

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"hypr-exiled/pkg/logger"
)

// initializeConfig creates or loads the configuration.
func initializeConfig(providedPath string, defaultPath string, log *logger.Logger) (*Config, error) {
	var config *Config
	var err error

	// Try provided path first if specified
	if providedPath != "" {
		config, err = loadConfigFromPath(providedPath, log)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from provided path: %w", err)
		}
	} else {
		// Try default path, create if doesn't exist
		if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
			config, err = DefaultConfig(log)
			if err != nil {
				return nil, err
			}

			data, err := json.MarshalIndent(config, "", "    ")
			if err != nil {
				return nil, err
			}

			if err := os.WriteFile(defaultPath, data, 0644); err != nil {
				return nil, err
			}
		} else {
			config, err = loadConfigFromPath(defaultPath, log)
			if err != nil {
				config, err = DefaultConfig(log)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return config, nil
}

// FindConfig locates and initializes the configuration.
func FindConfig(providedPath string, log *logger.Logger, embeddedAssets embed.FS) (*Config, error) {
	log.Info("Looking for configuration", "provided_path", providedPath)

	// Get user config directory
	homeConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Error("Failed to get user config directory", err)
		return nil, err
	}

	// Setup default paths
	defaultConfigDir := filepath.Join(homeConfigDir, "hypr-exiled")
	defaultConfigPath := filepath.Join(defaultConfigDir, "config.json")
	defaultLogsDir := filepath.Join(defaultConfigDir, "logs")

	log.Debug("Configuration paths",
		"config_dir", defaultConfigDir,
		"config_path", defaultConfigPath,
		"logs_dir", defaultLogsDir)

	// Create directory structure
	for _, dir := range []string{defaultConfigDir, defaultLogsDir} {
		log.Debug("Ensuring directory exists", "path", dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Error("Failed to create directory", err, "path", dir)
			return nil, err
		}
	}

	// Initialize config and load from appropriate source
	config, err := initializeConfig(providedPath, defaultConfigPath, log)
	if err != nil {
		return nil, err
	}

	// Setup assets once after config is loaded
	if err := config.setupAssets(defaultConfigDir, embeddedAssets); err != nil {
		return nil, err
	}

	return config, nil
}
