# Input Package

## Overview

Handles keyboard input automation for Path of Exile commands.

## Core Component

```go
type Input struct {
    windowManager *wm.Manager
    detector      *window.Detector
    log          *logger.Logger
    notifier     *notify.NotifyService
}
```

## Key Operations

### Command Execution
```go
ExecutePoECommands(commands []string) error
```

#### Process:
1. Verify PoE window active
2. Focus window
3. For each command:
   - Press Enter
   - Type command
   - Press Enter
   - Wait 50ms

### Timing Controls
- 100ms after window focus
- 50ms between keystrokes
- 50ms after command entry

## Best Practices

### Window Management
- Verify window state before input
- Handle focus failures gracefully
- Maintain window context

### Input Safety
- Command validation
- Proper key release
- Timing controls
- Error handling

### Dependencies
- robotgo: Keyboard automation
- window: PoE window detection
- wm: Window management
