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

// Config holds the application configuration.
type Config struct {
	// Configurable via JSON file (private fields to enforce immutability)
	poeLogPath    string
	triggers      map[string]string
	commands      map[string][]string
	notifyCommand string

	// Internal fields
	compiledTriggers map[string]*regexp.Regexp `json:"-"`
	log              *logger.Logger
	assetsDir        string `json:"-"`
}

// New creates a new Config instance with the provided logger.
func New(log *logger.Logger) *Config {
	return &Config{
		log: log,
	}
}

// LoadFromFile loads the configuration from a JSON file.
func (c *Config) LoadFromFile(path string, log *logger.Logger) error {
	log.Debug("Loading configuration from file", "path", path)

	data, err := os.ReadFile(path)
	if err != nil {
		log.Error("Failed to read config file", err, "path", path)
		return err
	}
	log.Debug("Config file read successfully", "size_bytes", len(data))

	// Use a temporary struct to unmarshal JSON
	var temp struct {
		PoeLogPath    string              `json:"poe_log_path"`
		Triggers      map[string]string   `json:"triggers"`
		Commands      map[string][]string `json:"commands"`
		NotifyCommand string              `json:"notify_command"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		log.Error("Failed to parse config JSON", err)
		return err
	}
	log.Debug("Config JSON parsed successfully")

	// Assign to private fields
	c.poeLogPath = temp.PoeLogPath
	c.triggers = temp.Triggers
	c.commands = temp.Commands
	c.notifyCommand = temp.NotifyCommand

	return c.compile()
}

// compile compiles the regex patterns in the triggers map.
func (c *Config) compile() error {
	log := c.log
	log.Debug("Compiling trigger patterns", "trigger_count", len(c.triggers))

	c.compiledTriggers = make(map[string]*regexp.Regexp)
	for name, pattern := range c.triggers {
		log.Debug("Compiling trigger pattern", "name", name, "pattern", pattern)

		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Error("Failed to compile trigger pattern", err, "name", name, "pattern", pattern)
			return err
		}
		c.compiledTriggers[name] = re
	}

	log.Debug("All trigger patterns compiled successfully", "compiled_count", len(c.compiledTriggers))
	return nil
}

// GetCommands returns a copy of the commands map.
func (c *Config) GetCommands() map[string][]string {
	commandsCopy := make(map[string][]string)
	for k, v := range c.commands {
		commandsCopy[k] = append([]string{}, v...) // Copy the slice
	}
	return commandsCopy
}

// GetTriggers returns a copy of the triggers map.
func (c *Config) GetTriggers() map[string]string {
	triggersCopy := make(map[string]string)
	for k, v := range c.triggers {
		triggersCopy[k] = v
	}
	return triggersCopy
}

// GetNotifyCommand returns the notify command.
func (c *Config) GetNotifyCommand() string {
	return c.notifyCommand
}

// GetPoeLogPath returns the PoE log path.
func (c *Config) GetPoeLogPath() string {
	return c.poeLogPath
}

// GetAssetsDir returns the assets directory.
func (c *Config) GetAssetsDir() string {
	return c.assetsDir
}

// getDefaultPoeLogPath finds the default Path of Exile 2 log file.
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

	log.Error("No valid POE log file found", nil, "checked_paths", possiblePaths)
	return "", fmt.Errorf("no valid Path of Exile 2 log file found in common locations")
}

// DefaultConfig creates a default configuration.
func DefaultConfig(log *logger.Logger) (*Config, error) {
	log.Debug("Creating default configuration")

	logPath, err := getDefaultPoeLogPath(log)
	if err != nil {
		log.Error("Failed to get default POE log path", err)
		return nil, fmt.Errorf("failed to get PoE log file, pls create the config file manually: %w", err)
	}

	config := &Config{
		poeLogPath: logPath,
		triggers: map[string]string{
			"incoming_trade": `\[INFO Client \d+\] @From ([^:]+): Hi, I would like to buy your ([^,]+(?:,[^,]+)*) listed for (\d+(?:\.\d+)?) ([^ ]+) in ([^\(]+) \(stash tab "([^"]+)"; position: left (\d+), top (\d+)\)`,
			"outgoing_trade": `\[INFO Client \d+\] @To ([^:]+): Hi, I would like to buy your ([^,]+(?:,[^,]+)*) listed for (\d+(?:\.\d+)?) ([^ ]+) in ([^\(]+) \(stash tab "([^"]+)"; position: left (\d+), top (\d+)\)`,
		},
		commands: map[string][]string{
			"party":  {"/invite {player}"},
			"finish": {"/kick {player}", "@{player} thanks!"},
			"trade":  {"/tradewith {player}"},
		},
		notifyCommand: "",
		log:           log,
	}

	log.Info("Created default configuration",
		"log_path", logPath,
		"trigger_count", len(config.triggers),
		"command_count", len(config.commands))

	if err := config.compile(); err != nil {
		log.Error("Failed to compile default config triggers", err)
		return nil, fmt.Errorf("failed to compile default config: %w", err)
	}

	return config, nil
}

// loadConfigFromPath loads the configuration from a file.
func loadConfigFromPath(path string, log *logger.Logger) (*Config, error) {
	config := &Config{log: log}
	if err := config.LoadFromFile(path, log); err != nil {
		return nil, err
	}
	return config, nil
}

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

// GetCurrencyIconPath returns the path to the currency icon.
func (c *Config) GetCurrencyIconPath(currencyType string) string {
	return filepath.Join(c.assetsDir, currencyType+".png")
}

// GetRofiThemePath returns the path to the Rofi theme.
func (c *Config) GetRofiThemePath() (string, error) {
	c.log.Debug("Getting Rofi theme path")
	themePath := filepath.Join(c.assetsDir, "trade.rasi")
	c.log.Debug("Using default Rofi theme", "path", themePath)
	return themePath, nil
}

// setupAssets sets up the assets directory and copies embedded assets.
func (c *Config) setupAssets(configDir string, embeddedAssets embed.FS) error {
	c.log.Debug("Setting up assets directory")

	// Set assets directory path
	c.assetsDir = filepath.Join(configDir, "assets")

	// Create assets directory if it doesn't exist
	if err := os.MkdirAll(c.assetsDir, 0755); err != nil {
		c.log.Error("Failed to create assets directory", err, "path", c.assetsDir)
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
		destFile := filepath.Join(c.assetsDir, entry.Name())

		// Check if file already exists
		if _, err := os.Stat(destFile); err == nil {
			c.log.Debug("Asset file exists, skipping", "file", destFile)
			continue
		}

		// Read embedded file
		data, err := embeddedAssets.ReadFile(sourceFile)
		if err != nil {
			c.log.Error("Failed to read embedded asset", err, "file", sourceFile)
			return fmt.Errorf("failed to read embedded asset %s: %w", sourceFile, err)
		}

		// Write to destination
		if err := os.WriteFile(destFile, data, 0644); err != nil {
			c.log.Error("Failed to write asset file", err, "destination", destFile)
			return fmt.Errorf("failed to write asset file %s: %w", destFile, err)
		}

		c.log.Debug("Copied asset file", "source", sourceFile, "destination", destFile)
	}

	c.log.Info("Assets setup completed", "assets_dir", c.assetsDir)
	return nil
}

func (c *Config) GetCompiledTriggers() map[string]*regexp.Regexp {
	triggersCopy := make(map[string]*regexp.Regexp)
	for k, v := range c.compiledTriggers {
		triggersCopy[k] = v
	}
	return triggersCopy
}
