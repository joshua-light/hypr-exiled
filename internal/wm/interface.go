package wm

type WindowManager interface {
	// FindWindow looks for a window by class name or title
	FindWindow(classNames []string, titles []string) (Window, error)
	// FocusWindow brings the specified window to front
	FocusWindow(Window) error
	// Name returns the WM name for logging/display
	Name() string
}

type Window struct {
	ID      string
	Class   string
	Title   string
	Address string // For Hyprland
}
