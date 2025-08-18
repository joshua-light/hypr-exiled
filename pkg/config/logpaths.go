package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"hypr-exiled/pkg/logger"
)

// ResolveLogPathForAppID chooses the correct client.txt by precedence
func (c *Config) ResolveLogPathForAppID(log *logger.Logger, appID int) (string, error) {
	name := c.GameNameByAppID(appID)

	if len(c.LogPaths) > 0 {
		if p, ok := c.LogPaths[strconv.Itoa(appID)]; ok && p != "" {
			if _, err := os.Stat(p); err == nil {
				log.Debug("Resolved log path via override", "app_id", appID, "path", p)
				return p, nil
			}
			return "", fmt.Errorf("configured log_paths[%d] does not exist", appID)
		}
		return "", fmt.Errorf("log_paths present but missing entry for appID %d (%s)", appID, name)
	}

	if base := c.poeLogPath; base != "" {
		if strings.Contains(base, name) {
			if _, err := os.Stat(base); err == nil {
				log.Debug("Using configured poe_log_path for current game", "app_id", appID, "path", base)
				return base, nil
			}
			return "", fmt.Errorf("configured poe_log_path does not exist")
		}
		for _, other := range []string{"Path of Exile 2", "Path of Exile"} {
			if other == name {
				continue
			}
			if strings.Contains(base, other) {
				candidate := strings.Replace(base, other, name, 1)
				if _, err := os.Stat(candidate); err == nil {
					log.Debug("Resolved log path via sibling swap", "from", other, "to", name, "path", candidate)
					return candidate, nil
				}
				return "", fmt.Errorf("derived sibling path not found: %s", candidate)
			}
		}
		return "", fmt.Errorf("poe_log_path set but cannot derive path for %s; add log_paths[%d] to config", name, appID)
	}

	p, err := GetDefaultPoeLogPathFor(log, name)
	if err != nil {
		return "", fmt.Errorf("default search failed for %s: %w", name, err)
	}
	log.Debug("Resolved log path via default search", "app_id", appID, "path", p)
	return p, nil
}
