package app

import (
	"encoding/json"
	"os"
	"regexp"
)

type Config struct {
	LogPath          string                    `json:"log_path"`
	Triggers         map[string]string         `json:"triggers"`
	Commands         map[string]string         `json:"commands"`
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

func DefaultConfig() *Config {
	config := &Config{
		LogPath: "", // Will be set based on OS
		Triggers: map[string]string{
			"trade": `.*@From ([^:]+):.*would like to buy`,
		},
		Commands: map[string]string{
			"trade":  "@trade {player}",
			"invite": "/invite {player}",
			"thank":  "@{player} thanks!",
		},
	}
	config.compile()
	return config
}
