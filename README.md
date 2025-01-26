# Hypr Exiled 🎮⚡

A lightweight Path of Exile 2 trade manager built for keyboard warriors and tiling WM enthusiasts.

## Why Choose Hypr Exiled? 🤔

Built for:

- **NixOS** and other AppImage-restricted distros 🐧
- **Hyprland** users wanting native integration 🪟
- Keyboard-driven workflows without mouse dependency ⌨️

> ℹ️ Prefer traditional GUIs? Check out [Exiled-Exchange-2](https://github.com/Kvan7/Exiled-Exchange-2) for AppImage builds

## How It Works 🔧

### Smart Architecture

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

### Key Benefits 🚀

- **Single Initialization**: All components initialized once in background service
- **Clean Separation**: Trade management vs direct actions vs UI layers
- **Consistent State**: One window manager connection for all operations
- **Easy Extensions**: Add new commands with just:
  1. IPC protocol update
  2. Handler function
  3. Client flag

## Get Started 🛠️

### Dependencies

Essential packages:

```bash
# Core
go rofi libX11 libXtst libXi libxcb

# Sound notifications
alsa-lib
```

### Build & Run

#### Using Nix Flakes

```bash
nix develop
go build -o hypr-exiled ./cmd/hypr-exiled
./hypr-exiled --debug  # Start service
```

#### Manual Build

Check flake.nix for required packages if building without Nix:

- Go 1.21+
- X11/XCB development headers
- Rofi
- ALSA development headers

### Essential Commands

```bash
# Start background service
./hypr-exiled

# Show trades UI (requires running service)
./hypr-exiled --showTrades

# Warp to hideout
./hypr-exiled --hideout
```

### Hyprland Keybinds

Add to your `hyprland.conf` (`hypr-exiled` background service must be running):

```bash
# Show trades UI when PoE 2 is focused (Mod+Shift+E)
bind = $mainMod SHIFT, E, exec, hyprctl activewindow | grep -q "class: steam_app_2694490" && /path/to/binary/hypr-exiled --showTrades

# Quick hideout when PoE 2 is focused (F5)
bind = , F5, exec, hyprctl activewindow | grep -q "class: steam_app_2694490" && /path/to/binary/hypr-exiled --hideout
```

## Core Features ✨

- Real-time trade monitoring 🔍
- Rofi-powered keyboard interface 🎨
- Cross-WM support (Hyprland/X11) 🖥️
- Automated trade responses 🤖

## Documentation 📚

### Architecture Overview

- [Main Application](cmd/hypr-exiled/DOC.MD): Entry point, service management
- [App Core](internal/app/DOC.MD): Application lifecycle, trade handling
- [IPC](internal/ipc/DOC.MD): Inter-process communication
- [POE Integration](internal/poe/DOC.MD): Game monitoring, window detection
- [Window Management](internal/wm/DOC.MD): WM abstraction layer
- [Trade Manager](internal/trade_manager/DOC.MD): Trade processing and UI
- [Input](internal/input/DOC.MD): Game input automation
- [Rofi](internal/rofi/DOC.MD): Trade UI implementation
- [Storage](internal/storage/DOC.MD): Trade data persistence
- [Notify](pkg/notify/DOC.MD): System notifications
- [Config](pkg/config/DOC.MD): Configuration management

## For Developers 👩‍💻

### Window Manager Support

Implement the interface:

```go
type WindowManager interface {
    FindWindow(classNames []string) (Window, error)
    FocusWindow(Window) error
    Name() string
}
```

### Development Environment (Nix)

```bash
nix develop  # Provides:
- Go 1.21+
- X11/XCB libs
- Rofi
- ALSA libs
- Development headers
```

## Contributing

[TODO: for now just follow the Architecture Overview sections]
See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.
