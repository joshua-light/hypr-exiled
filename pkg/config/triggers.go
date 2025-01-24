package config

import (
	"regexp"
)

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

// GetTriggers returns a copy of the triggers map.
func (c *Config) GetTriggers() map[string]string {
	triggersCopy := make(map[string]string)
	for k, v := range c.triggers {
		triggersCopy[k] = v
	}
	return triggersCopy
}

// GetCompiledTriggers returns a copy of the compiled triggers map.
func (c *Config) GetCompiledTriggers() map[string]*regexp.Regexp {
	triggersCopy := make(map[string]*regexp.Regexp)
	for k, v := range c.compiledTriggers {
		triggersCopy[k] = v
	}
	return triggersCopy
}
