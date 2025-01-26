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

# ipc/DOC.MD
```markdown
# IPC Package

## Overview
Unix domain socket-based communication system for command routing between UI and service components.

## Architecture

### Command Flow
```
                      +-------------------+
                      |  Background       |
                      |  Service          |
                      |                   |
                      |  (Initialized     |
                      |   Input/WM/Config)|
                      +-------------------+
                            ▲  ▲  ▲
                            |  |  |
            +---------------+  |  +-----------------+
            |                  |                    |
  +---------+----------+  +----+--------+  +--------+--------+
  | ./hypr-exiled      |  | ./hypr-exiled | | ./hypr-exiled   |
  | -showTrades        |  | -hideout      | | (background)    |
  +--------------------+  +---------------+ +-----------------+
```

## Components

### Protocol
```go
type Request struct {
    Command string `json:"command"`
}

type Response struct {
    Status  string `json:"status"`
    Message string `json:"message"`
}
```

### Supported Commands
- `showTrades`: Display trade UI
- `hideout`: Execute hideout command

### Socket Configuration
- Path: `/tmp/hypr-exiled.sock`
- Permissions: 0755
- Protocol: Unix domain socket

## Key Benefits

### Single Initialization
- Input and window manager initialized once
- No duplicate resource allocation
- Consistent state management

### Clean Separation
- TradeManager: Trade-specific logic
- Input: Direct PoE actions
- IPC: Command routing

### Maintainability
New commands require only:
1. IPC protocol update
2. Handler function
3. Client flag

# app/DOC.MD
```markdown
# HyprExiled Core Application

## Overview
Core service managing trade processing, window detection, and command execution through IPC.

## Key Components

### HyprExiled Struct
```go
type HyprExiled struct {
    entries       []models.TradeEntry
    poeLogWatcher *poe_log.LogWatcher
    TradeManager  *trade_manager.TradeManager
    detector      *window.Detector
    input         *input.Input
}
```

### Architecture Benefits
1. Single source of truth for all components
2. Centralized window management
3. Unified command handling
4. Clean resource lifecycle

### Key Operations
```go
NewHyprExiled()  // Initialize all components
Run()            // Start service and IPC server
Stop()           // Cleanup resources
```

## Implementation Details

### Initialization Flow
1. Dependency verification
2. Window detector setup
3. Input handler creation
4. Trade manager initialization
5. IPC server startup

### Service Lifecycle
1. Component initialization
2. IPC server startup
3. Log watcher activation
4. Signal handling
5. Graceful shutdown

### Error Handling
- Dependency validation
- Component initialization
- IPC communication
- Resource cleanup

### Best Practices
- Single component instances
- Structured logging
- Clean shutdown handling
- IPC-based communication
```
