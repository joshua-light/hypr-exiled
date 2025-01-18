package wm

import (
	"os/exec"
	"strings"
)

type X11 struct{}

func NewX11() (WindowManager, error) {
	// Check if xdotool is available
	if _, err := exec.LookPath("xdotool"); err != nil {
		return nil, err
	}
	return &X11{}, nil
}

func (x *X11) Name() string {
	return "X11"
}

func (x *X11) FindWindow(classNames []string, titles []string) (Window, error) {
	// Implementation using xdotool
	for _, class := range classNames {
		out, err := exec.Command("xdotool", "search", "--class", class).Output()
		if err == nil && len(out) > 0 {
			return Window{
				ID:    strings.TrimSpace(string(out)),
				Class: class,
			}, nil
		}
	}

	for _, title := range titles {
		out, err := exec.Command("xdotool", "search", "--name", title).Output()
		if err == nil && len(out) > 0 {
			return Window{
				ID:    strings.TrimSpace(string(out)),
				Title: title,
			}, nil
		}
	}

	return Window{}, nil
}

func (x *X11) FocusWindow(w Window) error {
	return exec.Command("xdotool", "windowactivate", w.ID).Run()
}
