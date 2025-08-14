package config

import "fmt"

type SteamAppSpec struct {
	Name        string `mapstructure:"name"        json:"name"`
	AppID       int    `mapstructure:"app_id"      json:"app_id"`
	WindowClass string `mapstructure:"window_class" json:"window_class"`
}

// fallback-registry, if nothing is specified in the config file
var defaultSteamApps = []SteamAppSpec{
	{Name: "Path of Exile", AppID: 238960, WindowClass: "steam_app_238960"},
	{Name: "Path of Exile 2", AppID: 2694490, WindowClass: "steam_app_2694490"},
}

func (c *Config) GetSteamApps() []SteamAppSpec {
	if c != nil && len(c.SteamApps) > 0 {
		return c.SteamApps
	}
	return defaultSteamApps
}

func (c *Config) GetDefaultAppID() int {
	if c != nil && c.DefaultAppID != 0 {
		return c.DefaultAppID
	}
	// default: PoE2
	return 2694490
}

func (c *Config) GameNameByAppID(id int) string {
	for _, a := range c.GetSteamApps() {
		if a.AppID == id {
			return a.Name
		}
	}
	return fmt.Sprintf("App %d", id)
}

func (c *Config) WindowClasses() []string {
	apps := c.GetSteamApps()
	out := make([]string, 0, len(apps))

	for _, a := range apps {
		if a.WindowClass != "" {
			out = append(out, a.WindowClass)
		} else {
			out = append(out, fmt.Sprintf("steam_app_%d", a.AppID))
		}
	}

	return out
}

func (c *Config) AppIDByWindowClass(class string) (int, bool) {
	for _, a := range c.GetSteamApps() {
		wc := a.WindowClass
		if wc == "" {
			wc = fmt.Sprintf("steam_app_%d", a.AppID)
		}
		if wc == class {
			return a.AppID, true
		}
	}
	return 0, false
}
