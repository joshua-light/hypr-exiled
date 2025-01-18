package wm

import (
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

type Hyprland struct{}

func NewHyprland() (*Hyprland, error) {
	// Check if hyprctl is available
	if _, err := exec.LookPath("hyprctl"); err != nil {
		return nil, err
	}
	return &Hyprland{}, nil
}

func (h *Hyprland) Name() string {
	return "Hyprland"
}

func (h *Hyprland) FindWindow(classNames []string, titles []string) (Window, error) {
	out, err := exec.Command("hyprctl", "clients", "-j").Output()
	if err != nil {
		return Window{}, err
	}

	var windows []struct {
		Address string `json:"address"`
		Class   string `json:"class"`
		Title   string `json:"title"`
	}
	if err := json.Unmarshal(out, &windows); err != nil {
		return Window{}, err
	}

	for _, w := range windows {
		for _, class := range classNames {
			if strings.Contains(strings.ToLower(w.Class), strings.ToLower(class)) {
				return Window{
					Class:   w.Class,
					Title:   w.Title,
					Address: w.Address,
				}, nil
			}
		}
		for _, title := range titles {
			if strings.Contains(strings.ToLower(w.Title), strings.ToLower(title)) {
				return Window{
					Class:   w.Class,
					Title:   w.Title,
					Address: w.Address,
				}, nil
			}
		}
	}

	return Window{}, nil
}

func (h *Hyprland) FocusWindow(w Window) error {
	exec.Command("hyprctl", "dispatch", "focuswindow", "address:"+w.Address).Run()
	time.Sleep(100 * time.Millisecond)
	return nil
}
