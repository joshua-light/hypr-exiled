package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// TODO: Add inf logs about config being loaded and from where
type Config struct {
	PoeLogPath       string                    `json:"poe_log_path"`
	Triggers         map[string]string         `json:"triggers"`
	Commands         map[string]string         `json:"commands"`
	NotifyCommand    string                    `json:"notify_command"`
	CompiledTriggers map[string]*regexp.Regexp `json:"-"`
}

func (c *Config) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, c); err != nil {
		return err
	}

	return c.compile()
}

func (c *Config) compile() error {
	c.CompiledTriggers = make(map[string]*regexp.Regexp)
	for name, pattern := range c.Triggers {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return err
		}
		c.CompiledTriggers[name] = re
	}
	return nil
}

func getDefaultPoeLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Common POE2 log paths
	possiblePaths := []string{
		filepath.Join(home, ".local", "share", "Steam", "steamapps", "common", "Path of Exile 2", "logs", "Client.txt"),
		filepath.Join(home, "Games", "Path of Exile 2", "logs", "Client.txt"),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no valid Path of Exile 2 log file found in common locations")
}

func DefaultConfig() (*Config, error) {
	logPath, err := getDefaultPoeLogPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get PoE log file, pls create the config file manually: %w", err)
	}

	config := &Config{
		PoeLogPath: logPath,
		Triggers: map[string]string{
			"trade": `\[INFO Client \d+\] @From ([^:]+):.*would like to buy`,
		},
		Commands: map[string]string{
			"trade":  "@trade {player}",
			"invite": "/invite {player}",
			"thank":  "@{player} thanks!",
		},
		NotifyCommand: "", // Empty by default
	}

	if err := config.compile(); err != nil {
		return nil, fmt.Errorf("failed to compile default config: %w", err)
	}

	return config, nil
}

// FindConfig looks for config in the following order:
// 1. Creates default config structure if it doesn't exist
// 2. Uses provided path if specified
// 3. Uses ~/.config/rofi-poe-helper/config.json
// 4. Falls back to default config if no file exists
func FindConfig(providedPath string) (*Config, error) {
	// Get user config directory
	homeConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	// Setup default paths
	defaultConfigDir := filepath.Join(homeConfigDir, "rofi-poe-helper")
	defaultConfigPath := filepath.Join(defaultConfigDir, "config.json")
	defaultLogsDir := filepath.Join(defaultConfigDir, "logs")

	// Create directory structure
	for _, dir := range []string{defaultConfigDir, defaultLogsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	// If default config doesn't exist, create it
	if _, err := os.Stat(defaultConfigPath); os.IsNotExist(err) {
		defaultConfig, err := DefaultConfig()
		if err != nil {
			return nil, err
		}
		data, err := json.MarshalIndent(defaultConfig, "", "    ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(defaultConfigPath, data, 0644); err != nil {
			return nil, err
		}
	}

	// 1. Try provided path if specified
	if providedPath != "" {
		config := &Config{}
		if err := config.LoadFromFile(providedPath); err == nil {
			return config, nil
		}
		// If provided path fails, return error instead of falling back
		return nil, fmt.Errorf("failed to load config from provided path: %w", err)
	}

	// 2. Try default config path
	config := &Config{}
	if err := config.LoadFromFile(defaultConfigPath); err == nil {
		return config, nil
	}

	// 3. Fall back to default config if everything else fails
	return DefaultConfig()
}
