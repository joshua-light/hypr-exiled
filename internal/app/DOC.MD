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

### Multi-Title (PoE1/PoE2) Support — Additions
- Config-driven Steam apps
- Initial log path resolution
- Hot watcher rotation
```
