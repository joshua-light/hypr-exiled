package config

// GetCommands returns a copy of the commands map.
func (c *Config) GetCommands() map[string][]string {
	commandsCopy := make(map[string][]string)
	for k, v := range c.commands {
		commandsCopy[k] = append([]string{}, v...) // Copy the slice
	}
	return commandsCopy
}

// GetNotifyCommand returns the notify command.
func (c *Config) GetNotifyCommand() string {
	return c.notifyCommand
}
