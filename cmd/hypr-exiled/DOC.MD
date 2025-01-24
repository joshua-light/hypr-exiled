# Hypr Exiled Documentation

## `main.go`

The `main.go` file is the entry point for the application. It:
1. Parses command-line arguments.
2. Initializes logging and configuration.
3. Handles the `--showTrades` command to display the trades UI.
4. Starts the background service if no specific command is provided.

---

### Key Features

- **Command-Line Arguments**:
  - `--config`: Path to the configuration file.
  - `--debug`: Enables debug logging.
  - `--showTrades`: Displays the trades UI.

- **Embedded Assets**:
  - Icons (`divine.png`, `exalt.png`) and a Rofi theme (`trade.rasi`) are embedded into the binary.

- **Background Service**:
  - Manages the application in the background, handling trades and notifications.

---

### Example Usage

```bash
# Start the background service
./hypr-exiled --config /path/to/config.json

# Show the trades UI
./hypr-exiled --showTrades

# Enable debug logging
./hypr-exiled --debug
```

### Code Overview

#### `main()`
- Parses flags and initializes logging.
- Calls `handleShowTrades` if `--showTrades` is provided.
- Otherwise, starts the background service using `startBackgroundService`.

#### `handleShowTrades()`
- Loads the configuration.
- Initializes global state (config, logger, and notifier).
- Sends a command to the background service to display trades.

#### `startBackgroundService()`
- Loads the configuration.
- Initializes global instances.
- Creates and starts the application.

### Best Practices

- **Logging**: Use the provided logger for all logging needs.
- **Configuration**: Always load the configuration before starting the application.
- **Error Handling**: Handle errors gracefully and log them appropriately.
- **Embedded Assets**: Use the embed package to include assets directly in the binary.
