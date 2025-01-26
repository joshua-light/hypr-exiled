# Input Package

## Overview
Handles keyboard input automation for Path of Exile commands through a unified IPC architecture.

## Key Architecture

### Command Flow
```
                   +---------------+
                   | IPC Command   |
                   +-------+-------+
                           |
                   +-------v-------+
                   | Input Handler |
                   +-------+-------+
                           |
                   +-------v-------+
                   | Window Manager
                   +---------------+
```

## Core Components

### Input Handler
```go
type Input struct {
    windowManager *wm.Manager
    detector      *window.Detector
    log          *logger.Logger
    notifier     *notify.NotifyService
}
```

### Key Operations

#### Command Methods
```go
ExecutePoECommands(commands []string) error  // Generic command execution
ExecuteHideout() error                       // Dedicated hideout command
```

### Process Flow
1. Command received via IPC
2. Window state verification
3. Focus management
4. Command execution
5. Response handling

### Error Handling
- Standardized error responses
- IPC-friendly error formats
- Window state validation
- Focus management errors

### Best Practices
- Single instance per service
- Centralized window management
- Consistent error handling
- IPC-friendly responses


