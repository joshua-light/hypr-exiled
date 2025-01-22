package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"hypr-exiled/pkg/logger"
)

type Config struct {
	PoeLogPath       string                    `json:"poe_log_path"`
	Triggers         map[string]string         `json:"triggers"`
	Commands         map[string]string         `json:"commands"`
	NotifyCommand    string                    `json:"notify_command"`
	CompiledTriggers map[string]*regexp.Regexp `json:"-"`
	log              *logger.Logger
}

func New(log *logger.Logger) *Config {
	return &Config{
		log: log,
	}
}

func (c *Config) LoadFromFile(path string, log *logger.Logger) error {
	log.Debug("Loading configuration from file",
		"path", path)

	data, err := os.ReadFile(path)
	if err != nil {
		log.Error("Failed to read config file", err,
			"path", path)
		return err
	}
	log.Debug("Config file read successfully",
		"size_bytes", len(data))

	if err := json.Unmarshal(data, c); err != nil {
		log.Error("Failed to parse config JSON", err)
		return err
	}
	log.Debug("Config JSON parsed successfully")

	return c.compile()
}

func (c *Config) compile() error {
	log := c.log
	log.Debug("Compiling trigger patterns",
		"trigger_count", len(c.Triggers))

	c.CompiledTriggers = make(map[string]*regexp.Regexp)
	for name, pattern := range c.Triggers {
		log.Debug("Compiling trigger pattern",
			"name", name,
			"pattern", pattern)

		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Error("Failed to compile trigger pattern", err,
				"name", name,
				"pattern", pattern)
			return err
		}
		c.CompiledTriggers[name] = re
	}

	log.Debug("All trigger patterns compiled successfully",
		"compiled_count", len(c.CompiledTriggers))
	return nil
}

func getDefaultPoeLogPath(log *logger.Logger) (string, error) {
	log.Debug("Looking for default POE log path")

	home, err := os.UserHomeDir()
	if err != nil {
		log.Error("Failed to get home directory", err)
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Common POE2 log paths
	possiblePaths := []string{
		filepath.Join(home, ".local", "share", "Steam", "steamapps", "common", "Path of Exile 2", "logs", "Client.txt"),
		filepath.Join(home, "Games", "Path of Exile 2", "logs", "Client.txt"),
	}

	for _, path := range possiblePaths {
		log.Debug("Checking possible log path", "path", path)
		if _, err := os.Stat(path); err == nil {
			log.Info("Found POE log file", "path", path)
			return path, nil
		}
	}

	log.Error("No valid POE log file found", nil,
		"checked_paths", possiblePaths)
	return "", fmt.Errorf("no valid Path of Exile 2 log file found in common locations")
}

func DefaultConfig(log *logger.Logger) (*Config, error) {
	log.Debug("Creating default configuration")

	logPath, err := getDefaultPoeLogPath(log)
	if err != nil {
		log.Error("Failed to get default POE log path", err)
		return nil, fmt.Errorf("failed to get PoE log file, pls create the config file manually: %w", err)
	}

	config := &Config{
		PoeLogPath: logPath,
		Triggers: map[string]string{
			// Buying from others (they want to sell to us)
			"incoming_trade": `\[INFO Client \d+\] @From ([^:]+): Hi, I would like to buy your ([^,]+(?:,[^,]+)*) listed for (\d+(?:\.\d+)?) ([^ ]+) in ([^\(]+) \(stash tab "([^"]+)"; position: left (\d+), top (\d+)\)`,
			// Selling to others (they want to buy from us)
			"outgoing_trade": `\[INFO Client \d+\] @To ([^:]+): Hi, I would like to buy your ([^,]+(?:,[^,]+)*) listed for (\d+(?:\.\d+)?) ([^ ]+) in ([^\(]+) \(stash tab "([^"]+)"; position: left (\d+), top (\d+)\)`,
		},
		Commands: map[string]string{
			"trade":  "@trade {player}",
			"invite": "/invite {player}",
			"thank":  "@{player} thanks!",
		},
		NotifyCommand: "",
		log:           log, // Initialize the logger field
	}

	log.Info("Created default configuration",
		"log_path", logPath,
		"trigger_count", len(config.Triggers),
		"command_count", len(config.Commands))

	if err := config.compile(); err != nil {
		log.Error("Failed to compile default config triggers", err)
		return nil, fmt.Errorf("failed to compile default config: %w", err)
	}

	return config, nil
}

func FindConfig(providedPath string, log *logger.Logger) (*Config, error) {
	log.Info("Looking for configuration",
		"provided_path", providedPath)

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

	// If default config doesn't exist, create it
	if _, err := os.Stat(defaultConfigPath); os.IsNotExist(err) {
		log.Info("Default config not found, creating new one",
			"path", defaultConfigPath)

		defaultConfig, err := DefaultConfig(log)
		if err != nil {
			log.Error("Failed to create default config", err)
			return nil, err
		}

		data, err := json.MarshalIndent(defaultConfig, "", "    ")
		if err != nil {
			log.Error("Failed to marshal default config", err)
			return nil, err
		}

		if err := os.WriteFile(defaultConfigPath, data, 0644); err != nil {
			log.Error("Failed to write default config file", err,
				"path", defaultConfigPath)
			return nil, err
		}

		log.Info("Created new default config file",
			"path", defaultConfigPath,
			"size_bytes", len(data))
	}

	// 1. Try provided path if specified
	if providedPath != "" {
		log.Debug("Attempting to load config from provided path",
			"path", providedPath)

		config := &Config{}
		if err := config.LoadFromFile(providedPath, log); err == nil {
			log.Info("Successfully loaded config from provided path",
				"path", providedPath)
			return config, nil
		}
		// If provided path fails, return error instead of falling back
		log.Error("Failed to load config from provided path", err,
			"path", providedPath)
		return nil, fmt.Errorf("failed to load config from provided path: %w", err)
	}

	// 2. Try default config path
	log.Debug("Attempting to load config from default path",
		"path", defaultConfigPath)

	config := &Config{
		log: log, // Initialize with logger
	}

	if err := config.LoadFromFile(defaultConfigPath, log); err == nil {
		log.Info("Successfully loaded config from default path",
			"path", defaultConfigPath)
		return config, nil
	}

	// 3. Fall back to default config if everything else fails
	log.Info("Falling back to default configuration")
	return DefaultConfig(log)
}
