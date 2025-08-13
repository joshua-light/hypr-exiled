package wm

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/notify"
)

type X11 struct {
	hasLoggedWaiting bool
	lastFoundWindow  Window
}

func NewX11() (WindowManager, error) {
	log := global.GetLogger()

	if _, err := exec.LookPath("xdotool"); err != nil {
		log.Error("xdotool not found in PATH", err)
		return nil, fmt.Errorf("xdotool is required for X11 support: %w", err)
	}

	return &X11{}, nil
}

func (x *X11) Name() string {
	return "X11"
}

func (x *X11) FindWindow(classNames []string) (Window, error) {
	log := global.GetLogger()
	notifier := global.GetNotifier()

	for _, class := range classNames {
		out, err := exec.Command("xdotool", "search", "--class", class).CombinedOutput()
		if err != nil {
			log.Debug("xdotool search failed", "class", class, "error", err)
			continue
		}

		windowIDs := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, windowID := range windowIDs {
			if windowID == "" {
				continue
			}

			classNameOut, err := exec.Command("xdotool", "getwindowclassname", windowID).CombinedOutput()
			if err != nil {
				log.Debug("Failed to get window class", "windowID", windowID, "error", err)
				continue
			}

			foundWindow := Window{
				Address: windowID,
				Class:   strings.TrimSpace(string(classNameOut)),
			}

			if foundWindow != x.lastFoundWindow {
				log.Debug("Found matching window",
					"class", foundWindow.Class,
					"address", foundWindow.Address)
				x.lastFoundWindow = foundWindow
			}

			x.hasLoggedWaiting = false
			return foundWindow, nil
		}
	}

	if x.lastFoundWindow != (Window{}) {
		x.lastFoundWindow = Window{}
	}

	if !x.hasLoggedWaiting {
		message := "Waiting for PoE Window..."
		log.Info(message)
		notifier.Show(message, notify.Info)
		x.hasLoggedWaiting = true
	}

	return Window{}, nil
}

func (x *X11) FocusWindow(w Window) error {
	log := global.GetLogger()

	log.Debug("Focusing X11 window", "address", w.Address, "class", w.Class)

	cmd := exec.Command("xdotool", "windowactivate", w.Address)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Error("Failed to focus window",
			err,
			"output", string(output),
			"address", w.Address)
		return fmt.Errorf("failed to focus window: %w", err)
	}

	time.Sleep(100 * time.Millisecond)
	return nil
}
