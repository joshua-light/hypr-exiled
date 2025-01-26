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
