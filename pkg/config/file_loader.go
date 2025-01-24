package config

import (
	"encoding/json"
	"os"

	"hypr-exiled/pkg/logger"
)

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

// loadConfigFromPath loads the configuration from a file.
func loadConfigFromPath(path string, log *logger.Logger) (*Config, error) {
	config := &Config{log: log}
	if err := config.LoadFromFile(path, log); err != nil {
		return nil, err
	}
	return config, nil
}
