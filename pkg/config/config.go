package config

import (
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

	//Steam AppIDs
	SteamApps    []SteamAppSpec `mapstructure:"steam_apps"    json:"steam_apps"`
	DefaultAppID int            `mapstructure:"default_app_id" json:"default_app_id"`
	// Optional per-AppID log path overrides (JSON keys are strings, e.g. "238960")
	LogPaths map[string]string `mapstructure:"log_paths"    json:"log_paths"`
}

// New creates a new Config instance with the provided logger.
func New(log *logger.Logger) *Config {
	return &Config{
		log: log,
	}
}

// GetPoeLogPath returns the PoE log path.
func (c *Config) GetPoeLogPath() string {
	return c.poeLogPath
}
