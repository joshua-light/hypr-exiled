# Notification Package

## Core Components

### NotifyService
```go
type NotifyService struct {
    log           *logger.Logger
    notifyCommand string
}

type NotificationType int
const (
    Error NotificationType = iota
    Info
)
```

## Notification Pipeline

1. **Custom Command**
   - Uses configured notify command
   - Formats: `command 'TYPE' 'TITLE' 'MESSAGE'`

2. **System Tools**
   ```go
   Tools := []string{
       "dunstify",    // Urgency: normal/critical
       "notify-send", // Urgency: normal/critical
       "zenity"       // Flags: --info/--error
   }
   ```

3. **Terminal Fallback**
   - Detects terminal environment
   - Supports: gnome-terminal, konsole, xfce4-terminal
   - Color-coded output (green/red)

4. **Log File**
   - Location: `~/.hypr-exiled-notifications.log`
   - Format: `[timestamp] Title - TYPE: Message`

## Terminal Detection

### Environment Variables
```
TERMINAL
TERMCMD
TERM_PROGRAM
DEFAULT_TERMINAL
KONSOLE_DBUS_SESSION
GNOME_TERMINAL_SCREEN
```

### Common Terminals
```
x-terminal-emulator
gnome-terminal
konsole
xfce4-terminal
alacritty
kitty
```

## Best Practices

- Graceful degradation through notification chain
- Color-coded messages by type
- Consistent formatting across methods
- Proper error context
- Resource cleanup
