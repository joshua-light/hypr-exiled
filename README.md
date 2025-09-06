# Hypr Exiled 🎮⚡

A lightweight Path of Exile trade manager built for keyboard power users and tiling WM's.

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


### Benefits 🚀

- **Single initialization**: All components initialized once in background service
- **Clean separation**: Trade management vs direct actions vs UI layers
- **Consistent state**: One window manager connection for all operations
- **Easy extensions**: Add new commands with just:
  1. IPC protocol update
  2. Handler function
  3. Client flag

## Installation & Usage Guide 🛠️

### System Requirements

### Hyprland

- Hyprland compositor
- rofi
- alsa-lib
- dunstify, notify-send, or zenity (for notifications)

### X11

- xdotool
- i3, bspwm, dwm, awesome, xmonad, etc
- rofi
- alsa-lib
- dunstify, notify-send, or zenity (for notifications)


## Quick Start Options

### Option 1: Download Verified Binary

1. Download the latest release from [GitHub Releases](https://github.com/gfsd3v/hypr-exiled/releases)
2. Verify the binary signature (recommended):
   ```bash
   curl -O https://raw.githubusercontent.com/gfsd3v/hypr-exiled/main/cosign.pub
   cosign verify-blob --key cosign.pub --signature checksums.txt.sig checksums.txt
   sha256sum -c checksums.txt
   ```
   > Binaries are signed with [Sigstore Cosign](https://docs.sigstore.dev)

### Option 2: Build from Source

Prerequisites:

- Go 1.21+
- X11/XCB development headers
- Rofi
- ALSA development headers

Using Nix Flakes:

```bash
nix develop
go build -o hypr-exiled ./cmd/hypr-exiled
```

Manual build:

```bash
# Install dependencies first
# Debian/Ubuntu
sudo apt install golang alsa-lib-devel rofi libx11-devel libxtst-devel libxi-devel libxcb-devel xdotool

# Arch Linux
sudo pacman -S go alsa-lib rofi libx11 libxtst libxi libxcb xdotool

# Build
go build -o hypr-exiled ./cmd/hypr-exiled
```

## Basic Usage

1. Start the background service:

   ```bash
   ./hypr-exiled --debug  # Debug mode for verbose logging
   ```

2. Available commands:
   ```bash
   ./hypr-exiled -showTrades  # Open trade UI
   ./hypr-exiled -hideout     # Warp to hideout
   ./hypr-exiled -search      # Search hovered item on PoE 2 trade site
   ```

## Window Manager Configuration

After everything is running, you can bind commands to your window manager keybindings.

### Hyprland

Add to your `~/.config/hypr/hyprland.conf`:

```bash
# Trade UI (Mod+Shift+E)
bind = $mainMod SHIFT, E, exec, hyprctl activewindow | grep -q "class: steam_app_238960" && /path/to/hypr-exiled -showTrades
bind = $mainMod SHIFT, E, exec, hyprctl activewindow | grep -q "class: steam_app_2694490" && /path/to/hypr-exiled -showTrades

# Quick hideout (F5)
bind = , F5, exec, hyprctl activewindow | grep -q "class: steam_app_238960" && /path/to/hypr-exiled -hideout
bind = , F5, exec, hyprctl activewindow | grep -q "class: steam_app_2694490" && /path/to/hypr-exiled -hideout

# Quick kingsmarch (F6)
bind = , F6, exec, hyprctl activewindow | grep -q "class: steam_app_238960" && /path/to/hypr-exiled -kingsmarch
bind = , F6, exec, hyprctl activewindow | grep -q "class: steam_app_2694490" && /path/to/hypr-exiled -kingsmarch

# Item search (F7)
bind = , F7, exec, hyprctl activewindow | grep -q "class: steam_app_238960" && /path/to/hypr-exiled -search
bind = , F7, exec, hyprctl activewindow | grep -q "class: steam_app_2694490" && /path/to/hypr-exiled -search
```

### i3wm

Add to your `~/.config/i3/config`:

```bash
# Trade UI
bindsym $mod+Shift+e exec --no-startup-id /path/to/hypr-exiled -showTrades

# Quick hideout
bindsym F5 exec --no-startup-id /path/to/hypr-exiled -hideout

# Quick kingsmarch
bindsym F6 exec --no-startup-id /path/to/hypr-exiled -kingsmarch

# Item search
bindsym F7 exec --no-startup-id /path/to/hypr-exiled -search
```

## PoE 2 Trade Site Integration 🛒

The new search feature allows you to quickly search for items on the official PoE 2 trade site:

1. **Hover over an item** in Path of Exile 2
2. **Press your configured hotkey** (default: F7)
3. The tool will:
   - Automatically copy the item to clipboard (Ctrl+C)
   - Parse the complete item tooltip (name, stats, quality, item level, etc.)
   - Construct an advanced search query with proper filters
   - Open your default browser with pre-filled search parameters on https://www.pathofexile.com/trade2

### Advanced Item Parsing

The tool performs comprehensive item analysis by extracting:

**Basic Item Information:**
- Item name and base type
- Rarity (Normal, Magic, Rare, Unique)
- Item level and quality percentage
- Corruption status and socket count

**Search Parameters:**
- Automatically constructs JSON query matching the official PoE 2 trade API format
- Includes item filters (quality, item level, corruption)  
- Uses appropriate search terms (unique name vs base type)
- Fallback to simple name search if parsing fails

**Smart Filtering:**
- Item level searches with ±5 level range for better results
- Quality filtering for enhanced items
- Corruption status filtering for exact matches
- Online-only listings by default

### Browser Compatibility

Works with any default browser on Linux via `xdg-open` command.


## Config file
Add config file under ~/.config/hypr-exile/config.json

```JSON
{
    "poe_log_path": "/your/SteamLibrary/steamapps/common/Path of Exile 2/logs/Client.txt",
    "log_paths": {
    "238960": "/your//SteamLibrary/steamapps/common/Path of Exile/logs/Client.txt",
    "2694490": "/your//SteamLibrary/steamapps/common/Path of Exile 2/logs/Client.txt"
    },
    "triggers": {
        "incoming_trade": "\\[INFO Client \\d+\\] @From (?:<[^>]+>\\s*)?(\\S+)\\s*: Hi, I would like to buy your ([^,]+(?:,[^,]+)*) listed for (\\d+(?:\\.\\d+)?) ([^ ]+) in ([^\\(]+) \\(stash tab \\\"([^\\\"]+)\\\"; position: left (\\d+), top (\\d+)\\)",
        "outgoing_trade": "\\[INFO Client \\d+\\] @To (?:<[^>]+>\\s*)?(\\S+)\\s*: Hi, I would like to buy your ([^,]+(?:,[^,]+)*) listed for (\\d+(?:\\.\\d+)?) ([^ ]+) in ([^\\(]+) \\(stash tab \\\"([^\\\"]+)\\\"; position: left (\\d+), top (\\d+)\\)"
    },
    "commands": {
        "party": [
            "/invite {player}"
        ],
        "finish": [
            "/kick {player}",
            "@{player} thanks!"
        ],
        "trade": [
            "/tradewith {player}"
        ]
    },
    "notifyCommand": ""
}
```

The "poe_log_path" key is used if the game is located in a different path then the default path.
If you have both games installed and they aren't in the default path you will need to add the game paths to the "log_paths" key.



## Troubleshooting

1. Use `--debug` flag for verbose logging
2. Ensure background service is running before using commands
3. Verify correct permissions on PoE log file
4. Check window manager integration (`xdotool` for `X11`, `hyprctl` for Hyprland)

## Core Features ✨

- Real-time trade monitoring 🔍
- Rofi-powered keyboard interface 🎨
- **PoE 2 Trade Site Integration**: Hover over items and press a hotkey to automatically search them on the official PoE 2 trade site 🛒
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
