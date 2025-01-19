package wm

import (
	"fmt"
	"os"
	"poe-helper/pkg/global"
)

// Manager handles window management operations based on the session type
type Manager struct {
	wm WindowManager
}

// NewManager creates a new window manager based on the session type
func NewManager() (*Manager, error) {
	log := global.GetLogger()

	// Check session type
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	log.Info("Session type detected", "session", sessionType)

	var wm WindowManager
	var err error

	switch sessionType {
	case "wayland":
		if sig := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE"); sig != "" {
			log.Debug("Initializing compositor support", "type", "Hyprland")
			wm, err = NewHyprland()
			if err != nil {
				return nil, fmt.Errorf("failed to initialize Hyprland support: %w", err)
			}
		} else {
			return nil, fmt.Errorf("unsupported Wayland compositor: only Hyprland is supported")
		}
	case "x11":
		log.Debug("Initializing compositor support", "type", "X11")
		wm, err = NewX11()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize X11 support: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported session type: %s", sessionType)
	}

	log.Info("Window manager initialized", "name", wm.Name())
	return &Manager{wm: wm}, nil
}

// FindWindow wraps the underlying window manager's FindWindow method
func (m *Manager) FindWindow(classNames []string, titles []string) (Window, error) {
	return m.wm.FindWindow(classNames, titles)
}

// FocusWindow wraps the underlying window manager's FocusWindow method
func (m *Manager) FocusWindow(w Window) error {
	return m.wm.FocusWindow(w)
}

// GetWMName returns the name of the current window manager
func (m *Manager) GetWMName() string {
	return m.wm.Name()
}
