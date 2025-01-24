# HyprExiled Core Application

## `app.go`

The `app.go` file implements the core application logic. It:
1. Manages trade entries and window detection
2. Handles Path of Exile log watching
3. Implements IPC server for trade management
4. Provides graceful shutdown handling

---

### Key Components

- **HyprExiled Struct**:
  - `entries`: List of processed trade entries
  - `poeLogWatcher`: Watches Path of Exile client.txt
  - `TradeManager`: Handles trade processing and UI
  - `detector`: Monitors game window state

- **Dependencies**:
  - Requires `rofi` for trade UI display
  - Uses IPC for communication between service and UI
  - Integrates with Path of Exile log file

---

### Main Functions

```go
// Create new instance with all components
NewHyprExiled()

// Start the service and watch for trades
Run()

// Gracefully shutdown all components
Stop()
```

### Implementation Details

#### `NewHyprExiled()`
- Verifies system dependencies
- Initializes window detector
- Sets up trade manager
- Creates log watcher instance

#### `Run()`
- Starts IPC socket server
- Initiates log watching
- Shows startup notification
- Waits for shutdown signal

#### `Stop()`
- Stops log watcher
- Cleans up resources
- Logs shutdown statistics

### Error Handling

- **Dependencies**: Checks for required system tools
- **Log Watcher**: Monitors for and reports watching errors
- **Notifications**: Reports critical errors to user
- **Graceful Shutdown**: Handles SIGINT and SIGTERM

### Best Practices

- **Logging**: Structured logging with debug/info levels
- **Error Context**: Detailed error information with metadata
- **Resource Management**: Proper cleanup on shutdown
- **Dependency Injection**: Uses global config and logger
