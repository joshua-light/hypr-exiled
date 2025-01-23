package config

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"hypr-exiled/pkg/logger"
)

type Config struct {
	// Configurable via use file
	PoeLogPath    string            `json:"poe_log_path"`
	Triggers      map[string]string `json:"triggers"`
	Commands      map[string]string `json:"commands"`
	NotifyCommand string            `json:"notify_command"`

	// Internal fields
	CompiledTriggers map[string]*regexp.Regexp `json:"-"`
	log              *logger.Logger
	AssetsDir        string `json:"-"`
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
		log:           log,
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

func loadConfigFromPath(path string, log *logger.Logger) (*Config, error) {
	config := &Config{log: log}
	if err := config.LoadFromFile(path, log); err != nil {
		return nil, err
	}
	return config, nil
}

// initializeConfig creates or loads the config
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

func (c *Config) GetCurrencyIconPath(currencyType string) string {
	return filepath.Join(c.AssetsDir, currencyType+".png")
}

func (c *Config) GetRofiThemePath() (string, error) {
	c.log.Debug("Getting Rofi theme path")
	themePath := filepath.Join(c.AssetsDir, "trade.rasi")
	c.log.Debug("Using default Rofi theme", "path", themePath)
	return themePath, nil
}

func (c *Config) setupAssets(configDir string, embeddedAssets embed.FS) error {
	c.log.Debug("Setting up assets directory")

	// Set assets directory path
	c.AssetsDir = filepath.Join(configDir, "assets")

	// Create assets directory if it doesn't exist
	if err := os.MkdirAll(c.AssetsDir, 0755); err != nil {
		c.log.Error("Failed to create assets directory", err, "path", c.AssetsDir)
		return fmt.Errorf("failed to create assets directory: %w", err)
	}

	// Copy embedded assets
	entries, err := embeddedAssets.ReadDir("assets")
	if err != nil {
		c.log.Error("Failed to read embedded assets", err)
		return fmt.Errorf("failed to read embedded assets: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		sourceFile := filepath.Join("assets", entry.Name())
		destFile := filepath.Join(c.AssetsDir, entry.Name())

		// Check if file already exists
		if _, err := os.Stat(destFile); err == nil {
			c.log.Debug("Asset file exists, skipping", "file", destFile)
			continue
		}

		// Read embedded file
		data, err := embeddedAssets.ReadFile(sourceFile)
		if err != nil {
			c.log.Error("Failed to read embedded asset", err,
				"file", sourceFile)
			return fmt.Errorf("failed to read embedded asset %s: %w", sourceFile, err)
		}

		// Write to destination
		if err := os.WriteFile(destFile, data, 0644); err != nil {
			c.log.Error("Failed to write asset file", err,
				"destination", destFile)
			return fmt.Errorf("failed to write asset file %s: %w", destFile, err)
		}

		c.log.Debug("Copied asset file",
			"source", sourceFile,
			"destination", destFile)
	}

	c.log.Info("Assets setup completed",
		"assets_dir", c.AssetsDir)

	return nil
}
