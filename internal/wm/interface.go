package wm

// WindowManager interface defines the required methods for window managers
type WindowManager interface {
	// FindWindow looks for a window by class name
	FindWindow(classNames []string) (Window, error)
	// FocusWindow brings the specified window to front
	FocusWindow(Window) error
	// Name returns the WM name for logging/display
	Name() string
}

// Window represents a window in the system
type Window struct {
	ID      string // For X11
	Class   string
	Address string // For Hyprland
}

// IsEmpty checks if the window is empty/not found
func (w Window) IsEmpty() bool {
	return w.ID == "" && w.Address == "" && w.Class == ""
}
