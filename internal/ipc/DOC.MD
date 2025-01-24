# IPC Package Documentation

## Overview

The IPC package implements Unix domain socket-based communication between the background service and UI components.

---

### Components

- **Client** (`client.go`):
  - Sends commands to the background service
  - Handles response processing
  - Manages socket connections

- **Server** (`server.go`):
  - Listens for client connections
  - Processes incoming commands
  - Manages trade display requests

---

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

### Socket Configuration
- Path: `/tmp/hypr-exiled.sock`
- Permissions: 0755
- Protocol: Unix domain socket

---

### Client Functions

#### `SendCommand(command string) (Response, error)`
- Establishes socket connection
- Sends JSON-encoded command
- Returns parsed response or error
- Handles connection cleanup

### Server Functions

#### `StartSocketServer(tradeManager *trade_manager.TradeManager)`
- Initializes socket listener
- Creates socket directory
- Handles incoming connections
- Spawns connection handlers

#### `handleConnection(conn net.Conn, tradeManager *trade_manager.TradeManager)`
- Decodes incoming requests
- Routes commands to trade manager
- Sends JSON-encoded responses
- Implements connection lifecycle

### Supported Commands

- `showTrades`: Displays trade UI
  - Success: Shows trade interface
  - Error: Returns error message

### Error Handling

- Socket initialization errors
- Connection failures
- JSON encoding/decoding errors
- Invalid commands
- Trade manager errors

### Best Practices

- Proper socket cleanup on startup
- Graceful connection handling
- Structured logging
- Comprehensive error messaging
- Goroutine-based connection handling
