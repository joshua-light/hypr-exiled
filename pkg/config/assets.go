package config

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

// GetAssetsDir returns the assets directory.
func (c *Config) GetAssetsDir() string {
	return c.assetsDir
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
