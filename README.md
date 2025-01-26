# Hypr Exiled 🎮⚡

A lightweight Path of Exile 2 trade manager built for keyboard warriors and tiling WM enthusiasts.


https://github.com/user-attachments/assets/5ea48204-d9b2-4690-8db5-b96446b869f4

## Who/What this is for

- **NixOS** and other AppImage-restricted distros 🐧
- **Hyprland** users wanting native integration 🪟
- **X11 Window Managers** with xdotool support:
  - i3
  - bspwm
  - dwm
  - awesome
  - xmonad
- Keyboard-driven workflows without mouse dependency ⌨️

> 📝 X11 support requires `xdotool` package installed

> ℹ️ Prefer traditional GUIs? Check out [Exiled-Exchange-2](https://github.com/Kvan7/Exiled-Exchange-2) for AppImage builds

### Benefits 🚀

- **Single initialization**: All components initialized once in background service
- **Clean separation**: Trade management vs direct actions vs UI layers
- **Consistent state**: One window manager connection for all operations
- **Easy extensions**: Add new commands with just:
  1. IPC protocol update
  2. Handler function
  3. Client flag

## Get Started 🛠️

### Dependencies

Essential packages:

```bash
# Core
go alsa-lib rofi libX11 libXtst libXi libxcb xdotool # xdotool needed for X11 WMs
```

### Build & Run

#### Using Nix Flakes

```bash
nix develop
go build -o hypr-exiled ./cmd/hypr-exiled
./hypr-exiled --debug  # Start service
```

#### Manual build

Check `flake.nix` for required packages if building without Nix:

- Go 1.21+
- X11/XCB development headers
- Rofi
- ALSA development headers

### Essential commands

```bash
# Start background service (requires to be running for any command to work)
./hypr-exiled

# Show trades UI
./hypr-exiled -showTrades

# Warp to hideout
./hypr-exiled -hideout
```

### Configuration

Default config path: `$HOME/.config/hypr-exiled/config.json`

```json
{
  "poe_log_path": "/path/to/steam/Path of Exile 2/logs/Client.txt",
  "notify_command": "dunstify", // Supports: dunstify, notify-send, zenity
  "triggers": {
    "incoming_trade": "...", // Trade message patterns
    "outgoing_trade": "..."
  },
  "commands": {
    "finish": ["/kick {player}", "@{player} thanks!"],
    "party": ["/invite {player}"],
    "trade": ["/tradewith {player}"]
  }
}
```

Override location: `--config /path/to/config.json`

### Hyprland Keybinds

Add to your `hyprland.conf` (`hypr-exiled` background service must be running):

```bash
# Show trades UI when PoE 2 is focused (Mod+Shift+E)
bind = $mainMod SHIFT, E, exec, hyprctl activewindow | grep -q "class: steam_app_2694490" && /path/to/binary/hypr-exiled -showTrades

# Quick hideout when PoE 2 is focused (F5)
bind = , F5, exec, hyprctl activewindow | grep -q "class: steam_app_2694490" && /path/to/binary/hypr-exiled -hideout
```

## Core Features ✨

- Real-time trade monitoring 🔍
- Rofi-powered keyboard interface 🎨
- Theoretical X11 support (untested) via `xdotool`:
  - Should work on common X11 distributions (Arch, Debian, Ubuntu, Fedora)
  - Compatible with tiling WMs like i3, bspwm, dwm, awesome, xmonad
  - Requires `xdotool` package installed
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

## For developers 👩‍💻

### Architecture

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

### WM Support

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

- [TODO: for now just follow the Architecture Overview sections]

## License

MIT License - see [LICENSE](LICENSE) for details.
