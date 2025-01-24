# Window Management Package

## Overview

Handles window management operations across different desktop environments (X11, Hyprland).

## Components

### Core Interface (`interface.go`)
```go
type WindowManager interface {
    FindWindow(classNames []string) (Window, error)
    FocusWindow(Window) error
    Name() string
}
```

### Manager (`manager.go`)
- Detects session type
- Initializes appropriate WM
- Provides unified interface

### Implementations
- **X11**: Uses xdotool
- **Hyprland**: Uses hyprctl

## Adding New WM Support

1. **Create Implementation File**
```go
package wm

type NewWM struct {
    // WM-specific fields
}

func NewNewWM() (*NewWM, error) {
    // Initialize WM
    // Check dependencies
    return &NewWM{}, nil
}
```

2. **Implement Interface Methods**
```go
// Return WM name for logging
func (w *NewWM) Name() string {
    return "NewWM"
}

// Find window by class names
func (w *NewWM) FindWindow(classNames []string) (Window, error) {
    // Implement window search
    // Return Window struct with appropriate fields
}

// Focus specific window
func (w *NewWM) FocusWindow(w Window) error {
    // Implement window focus
    // Add small delay after focus
}
```

3. **Update Manager**
```go
func NewManager() (*Manager, error) {
    switch sessionType {
    case "new_session_type":
        wm, err = NewNewWM()
    }
}
```

4. **Required Features**
- Window class matching
- Window focus capability
- Error handling for missing dependencies
- Session type detection
- Proper cleanup/resource management
- Logging integration
- Thread safety if needed

## Best Practices

### Window Detection
- Case-insensitive class matching
- Clean error messages
- Graceful handling of missing windows

### Window Focus
- Verify window exists before focus
- Add delay after focus (100ms typical)
- Handle focus errors gracefully

### Error Handling
- Check dependencies on startup
- Validate window IDs/addresses
- Provide context in error messages

### Logging
- Log WM initialization
- Debug logs for window operations
- Error details for failures
