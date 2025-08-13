package wm

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/notify"
)

type Hyprland struct {
	hasLoggedWaiting bool
	lastFoundWindow  Window
}

func NewHyprland() (*Hyprland, error) {
	log := global.GetLogger()

	// Check if hyprctl is available
	path, err := exec.LookPath("hyprctl")
	if err != nil {
		log.Error("hyprctl not found in PATH", err)
		return nil, fmt.Errorf("hyprctl not found in PATH: %w", err)
	}
	log.Debug("Found hyprctl", "path", path)

	return &Hyprland{}, nil
}

func (h *Hyprland) Name() string {
	return "Hyprland"
}

func (h *Hyprland) FindWindow(classNames []string) (Window, error) {
	log := global.GetLogger()
	notifier := global.GetNotifier()

	// Create the command
	cmd := exec.Command("hyprctl", "clients", "-j")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("Failed to execute hyprctl", err, "output", string(output))
		return Window{}, fmt.Errorf("hyprctl error: %w", err)
	}

	if len(output) == 0 {
		return Window{}, nil
	}

	var windows []struct {
		Address string `json:"address"`
		Class   string `json:"class"`
	}

	if err := json.Unmarshal(output, &windows); err != nil {
		log.Error("Failed to parse hyprctl output", err, "output", string(output))
		return Window{}, fmt.Errorf("failed to parse hyprctl output: %w", err)
	}

	// Search for matching window
	for _, w := range windows {
		// Check classes
		for _, class := range classNames {
			if strings.Contains(strings.ToLower(w.Class), strings.ToLower(class)) {
				foundWindow := Window{
					Class:   w.Class,
					Address: w.Address,
				}

				// Only log if this is a different window than last time
				if foundWindow != h.lastFoundWindow {
					log.Debug("Found matching window by class",
						"class", w.Class,
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
		var message = "Waiting for PoE Window..."
		log.Info(message)
		notifier.Show(message, notify.Info)
		h.hasLoggedWaiting = true
	}

	return Window{}, nil
}

func (h *Hyprland) FocusWindow(w Window) error {
	log := global.GetLogger()

	log.Debug("Focusing window", "address", w.Address)

	cmd := exec.Command("hyprctl", "dispatch", "focuswindow", "address:"+w.Address)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Error("Failed to focus window", err, "output", string(output))
		return fmt.Errorf("failed to focus window: %w", err)
	}

	time.Sleep(100 * time.Millisecond)
	return nil
}
