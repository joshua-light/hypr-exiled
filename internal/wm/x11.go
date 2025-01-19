package wm

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type X11 struct{}

func NewX11() (WindowManager, error) {
	// Check if xdotool is available
	if _, err := exec.LookPath("xdotool"); err != nil {
		return nil, fmt.Errorf("xdotool is required for X11 support but was not found: %w", err)
	}
	return &X11{}, nil
}

func (x *X11) Name() string {
	return "X11"
}

func (x *X11) FindWindow(classNames []string, titles []string) (Window, error) {
	for _, class := range classNames {
		out, err := exec.Command("xdotool", "search", "--class", class).Output()
		if err == nil && len(out) > 0 {
			// Get the first window ID (first line)
			windowID := strings.Split(strings.TrimSpace(string(out)), "\n")[0]

			// Get window title
			titleOut, err := exec.Command("xdotool", "getwindowname", windowID).Output()
			if err == nil {
				return Window{
					ID:    windowID,
					Class: class,
					Title: strings.TrimSpace(string(titleOut)),
				}, nil
			}
		}
	}

	for _, title := range titles {
		out, err := exec.Command("xdotool", "search", "--name", title).Output()
		if err == nil && len(out) > 0 {
			// Get the first window ID (first line)
			windowID := strings.Split(strings.TrimSpace(string(out)), "\n")[0]

			// Get window class
			classOut, err := exec.Command("xdotool", "getwindowclassname", windowID).Output()
			if err == nil {
				return Window{
					ID:    windowID,
					Class: strings.TrimSpace(string(classOut)),
					Title: title,
				}, nil
			}
		}
	}

	return Window{}, nil
}

func (x *X11) FocusWindow(w Window) error {
	if w.ID == "" {
		return fmt.Errorf("cannot focus window: no window ID provided")
	}

	err := exec.Command("xdotool", "windowactivate", w.ID).Run()
	if err != nil {
		return fmt.Errorf("failed to focus window: %w", err)
	}

	// Small delay to ensure window is focused
	time.Sleep(100 * time.Millisecond)
	return nil
}
