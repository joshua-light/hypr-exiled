# PoE Helper Documentation

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Installation](#installation)
4. [Development](#development)
5. [Testing](#testing)
6. [Logging](#logging)
7. [Troubleshooting](#troubleshooting)

## Overview

PoE Helper is a Go application designed to enhance Path of Exile 2 gameplay by providing automated responses to in-game events. It monitors the game's log file and provides a simple GUI for executing commands.

### Key Features

- Real-time log monitoring
- Window manager agnostic design
- Automated command execution
- Cross-platform input simulation
- Extensive logging for debugging

## Architecture

### Component Overview

```
poe-helper/
├── cmd/
│   └── poe-helper/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   └── config.go
│   ├── wm/
│   │   ├── interface.go
│   │   ├── hyprland.go
│   │   └── x11.go
│   └── input/
│       └── simulator.go
├── pkg/
│   └── logger/
│       └── logger.go
├── test/
│   └── fixtures/
│       └── sample_logs/
└── docs/
    └── images/
```

### Key Components

#### Window Manager Interface

The `wm.WindowManager` interface provides abstraction for different window managers:

```go
type WindowManager interface {
    FindWindow(classNames []string, titles []string) (Window, error)
    FocusWindow(Window) error
    Name() string
}
```

#### Input Simulation

Input simulation is handled through a platform-agnostic interface that supports both X11 and Wayland environments.

#### Logging System

The application uses structured logging with different log levels and outputs:

- File logging for debugging
- Console logging for development
- Structured JSON logging for production

## Installation

### Prerequisites

- Go 1.21 or higher
- wtype (input simulation)
- xdotool (optional, for X11 support)

### Build from Source

```bash
# Clone repository
git clone https://github.com/yourusername/poe-helper
cd poe-helper

# Install dependencies
go mod download

# Build
go build -o poe-helper cmd/poe-helper/main.go

# Run
./poe-helper
```

### Development Setup

1. Install development tools:

```bash
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

2. Set up pre-commit hooks:

```bash
#!/bin/sh
go fmt ./...
golangci-lint run
go test ./...
```

## Development

### Code Structure

- `cmd/`: Application entrypoints
- `internal/`: Private application code
- `pkg/`: Public libraries
- `test/`: Test files and fixtures

### Logging

The application uses different log levels:

- `DEBUG`: Detailed debugging information
- `INFO`: General operational information
- `WARN`: Warning messages
- `ERROR`: Error conditions
- `FATAL`: Critical errors that require shutdown

Example log setup:

```go
logger := log.NewLogger(
    log.WithFile("poe-helper.log"),
    log.WithConsole(),
    log.WithLevel(log.DebugLevel),
)
```

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Categories

1. Unit Tests: Test individual components
2. Integration Tests: Test component interactions
3. E2E Tests: Test complete workflows

## Troubleshooting

### Common Issues

1. Window Detection Issues

```
ERROR: Failed to detect PoE2 window
Solution: Check window class/title names using:
- Hyprland: hyprctl clients -j
- X11: xwininfo -root -tree
```

2. Input Simulation Issues

```
ERROR: Failed to simulate input
Solution: Verify wtype installation and permissions
```

### Debugging

Enable debug logging:

```bash
POE_HELPER_LOG_LEVEL=debug ./poe-helper
```

View logs:

```bash
tail -f ~/.local/share/poe-helper/logs/debug.log
```

## Contributing

### Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit pull request

### Code Style

Follow Go standard practices:

- Use `gofmt`
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Add comments for exported functions
