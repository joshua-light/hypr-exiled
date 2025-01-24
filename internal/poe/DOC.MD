# POE Package Documentation

## Overview

The POE package handles Path of Exile client.txt log monitoring and window detection.

---

### Log Watcher (`log/watcher.go`)

- **Purpose**: Monitors POE client.txt for trade messages
- **Key Components**:
  - Timestamp validation
  - Trade message parsing
  - Window state verification

```go
type LogWatcher struct {
    handler     func(models.TradeEntry)
    windowCheck *window.Detector
    stopChan    chan struct{}
    mu          sync.Mutex
    stopped     bool
}
```

#### Core Functions

- `NewLogWatcher()`: Creates watcher instance
- `Watch()`: Starts log monitoring
- `processLogLine()`: Parses trade messages
- `Stop()`: Graceful shutdown

#### Features
- Continuous log file monitoring
- File truncation handling
- Thread-safe operations
- Efficient line processing
- Trade entry validation

---

### Window Detector (`window/detect.go`)

- **Purpose**: Tracks POE window state
- **Key Components**:
  - Window activity monitoring
  - Session state tracking
  - Trade message validation

```go
type Detector struct {
    hyprExiledSessionStart time.Time
    lastResetTimestamp     time.Time
    windowFoundTime        time.Time
    isWindowActive         bool
    currentWindow          wm.Window
    // ... other fields
}
```

#### Core Functions

- `NewDetector()`: Creates detector instance
- `Detect()`: Checks window state
- `CheckLogLineValidity()`: Validates trade messages
- `Start()/Stop()`: Lifecycle management

#### Features
- Active window monitoring
- Game restart detection
- Session state tracking
- Thread-safe operations

### Message Processing

#### Trade Message Validation
- Timestamp check
- Window activity verification
- Session state validation
- Game restart handling

#### Message Types
- `@From`: Incoming trades
- `@To`: Outgoing trades

### Best Practices

- **Logging**: Structured logging throughout
- **Error Handling**: Comprehensive error reporting
- **Thread Safety**: Mutex-based synchronization
- **Resource Management**: Proper cleanup
- **Performance**: Efficient file reading
