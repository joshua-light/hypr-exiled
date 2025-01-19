package wm

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"poe-helper/pkg/logger"
	"strings"
	"time"
)

type Hyprland struct {
	log              *logger.Logger
	hasLoggedWaiting bool
	lastFoundWindow  Window
}

func NewHyprland(log *logger.Logger) (*Hyprland, error) {
	// Check if hyprctl is available
	path, err := exec.LookPath("hyprctl")
	if err != nil {
		log.Error("hyprctl not found in PATH", err)
		return nil, fmt.Errorf("hyprctl not found in PATH: %w", err)
	}
	log.Debug("Found hyprctl", "path", path)

	return &Hyprland{log: log}, nil
}

func (h *Hyprland) Name() string {
	return "Hyprland"
}

func (h *Hyprland) FindWindow(classNames []string, titles []string) (Window, error) {
	// Create the command
	cmd := exec.Command("hyprctl", "clients", "-j")
	output, err := cmd.CombinedOutput()
	if err != nil {
		h.log.Error("Failed to execute hyprctl", err, "output", string(output))
		return Window{}, fmt.Errorf("hyprctl error: %w", err)
	}

	if len(output) == 0 {
		return Window{}, nil
	}

	var windows []struct {
		Address string `json:"address"`
		Class   string `json:"class"`
		Title   string `json:"title"`
	}

	if err := json.Unmarshal(output, &windows); err != nil {
		h.log.Error("Failed to parse hyprctl output", err, "output", string(output))
		return Window{}, fmt.Errorf("failed to parse hyprctl output: %w", err)
	}

	// Search for matching window
	for _, w := range windows {
		// Check classes
		for _, class := range classNames {
			if strings.Contains(strings.ToLower(w.Class), strings.ToLower(class)) {
				foundWindow := Window{
					Class:   w.Class,
					Title:   w.Title,
					Address: w.Address,
				}

				// Only log if this is a different window than last time
				if foundWindow != h.lastFoundWindow {
					h.log.Debug("Found matching window by class",
						"class", w.Class,
						"title", w.Title,
						"address", w.Address)
					h.lastFoundWindow = foundWindow
				}

				h.hasLoggedWaiting = false
				return foundWindow, nil
			}
		}
		// Check titles
		for _, title := range titles {
			if strings.Contains(strings.ToLower(w.Title), strings.ToLower(title)) {
				foundWindow := Window{
					Class:   w.Class,
					Title:   w.Title,
					Address: w.Address,
				}

				// Only log if this is a different window than last time
				if foundWindow != h.lastFoundWindow {
					h.log.Debug("Found matching window by title",
						"class", w.Class,
						"title", w.Title,
						"address", w.Address)
					h.lastFoundWindow = foundWindow
				}

				h.hasLoggedWaiting = false
				return foundWindow, nil
			}
		}
	}

	// Reset last found window when no window is found
	if h.lastFoundWindow != (Window{}) {
		h.lastFoundWindow = Window{}
	}

	if !h.hasLoggedWaiting {
		h.log.Info("Waiting for PoE 2...")
		h.hasLoggedWaiting = true
	}

	return Window{}, nil
}

func (h *Hyprland) FocusWindow(w Window) error {
	h.log.Debug("Focusing window", "address", w.Address)

	cmd := exec.Command("hyprctl", "dispatch", "focuswindow", "address:"+w.Address)
	if output, err := cmd.CombinedOutput(); err != nil {
		h.log.Error("Failed to focus window", err, "output", string(output))
		return fmt.Errorf("failed to focus window: %w", err)
	}

	time.Sleep(100 * time.Millisecond)
	return nil
}
