# Hypr Exiled

A Path of Exile 2 trade manager designed for Hyprland, with support for X11 environments.

## Architecture

Hypr Exiled operates as a client-server application:

- **Background Service**: Monitors PoE logs and manages trades
- **Trade UI**: Rofi-based interface for interacting with trades

### Operation Flow

1. Start background service: `hypr-exiled`
2. Service monitors PoE logs for trades
3. Access trade UI: `hypr-exiled --showTrades`
4. UI communicates with service via Unix socket

## Features

- Real-time trade monitoring and management
- Rofi-based trade interface
- Automated trade responses
- Cross-window-manager support (Hyprland/X11)

## Dependencies

- `rofi`: Trade UI display
- `libX11`, `libXtst`, `libXi`, `libxcb`: Input simulation (required even on Wayland)
- `go` 1.21+

## Building and Running

### Build

```bash
# Using Nix Flakes (recommended)
# Make sure u have nix and nix flakes enabled
nix develop
go build -o hypr-exiled ./cmd/hypr-exiled

# Manual Build
go build -o hypr-exiled ./cmd/hypr-exiled
```

### Running

1. Start background service:

```bash
./hypr-exiled
```

2. Show trade UI (requires service running):

```bash
./hypr-exiled --showTrades
```

3. Enable debug logging:

```bash
./hypr-exiled --debug
```

Note: The background service must be running before using the `--showTrades` command. The service communicates with the UI through a Unix socket at `/tmp/hypr-exiled.sock`.

## Documentation

See individual module documentation for detailed information:

- [Main Application](doc/main.md): Entry point and service management
- [App Core](doc/app.md): Application lifecycle and trade handling
- [IPC](doc/ipc.md): Inter-process communication
- [POE Integration](doc/poe.md): Game log monitoring and window detection
- [Window Management](doc/wm.md): Window manager abstraction
- [Trade Manager](doc/trade-manager.md): Trade processing and UI
- [Input](doc/input.md): Game input automation
- [Rofi](doc/rofi.md): Trade UI implementation
- [Storage](doc/storage.md): Trade data persistence
- [Notify](doc/notify.md): System notifications

## Window Manager Support

Currently supports:

- Hyprland (primary focus)
- X11 (secondary support)

Adding support for new window managers requires implementing the `WindowManager` interface:

```go
type WindowManager interface {
    FindWindow(classNames []string) (Window, error)
    FocusWindow(Window) error
    Name() string
}
```

## Development Environment

The included `flake.nix` provides all necessary dependencies:

```nix
nix develop
```

### Included Development Tools

- Go toolchain
- X11/XCB libraries
- Rofi
- Required development headers

## Contributing

1. Ensure all dependencies are installed
2. Follow existing module documentation patterns
3. Implement tests for new features
4. Update relevant documentation
