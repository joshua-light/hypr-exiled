# Hypr-Exiled Agent Guide

## Project Overview

Hypr-Exiled is a Path of Exile 2 automation tool that provides various utilities for the game, including:

- **Trade Management**: Show trades UI and handle trading workflows
- **Quick Navigation**: Fast travel to hideout and Kingsmarch
- **Item Search**: Parse items from clipboard and open PoE 2 trade site with advanced search parameters
- **Price Checking**: API-based price analysis with stat filtering and market data

The application runs as a background service with IPC communication for command execution.

## Main Module Locations

### Core Application
- `cmd/hypr-exiled/main.go` - Entry point with CLI argument parsing and command routing
- `internal/app/` - Main application service logic
- `internal/ipc/` - Unix socket server for inter-process communication

### Input Handling
- `internal/input/input.go` - **Primary module** containing:
  - Item parsing and data extraction
  - Trade API query building
  - Price checking functionality
  - Search URL construction
  - Window management and automation

### Supporting Modules
- `internal/input/statsmap/` - External stat ID mapping from Exiled-Exchange-2
- `internal/poe/window/` - PoE window detection and management  
- `internal/wm/` - Window manager integration
- `internal/trade_manager/` - Trade workflow management
- `pkg/config/` - Configuration management
- `pkg/global/` - Global state and services
- `pkg/logger/` - Logging utilities
- `pkg/notify/` - Notification service

## Build Instructions

To build the project:

```bash
go build -o hypr-exiled ./cmd/hypr-exiled
```

The executable will be created as `hypr-exiled` in the current directory.

## Requirements

- Go 1.21+
- Linux environment (uses xdg-open, Unix sockets)
- POESESSID environment variable for trade API access

## Agent Guidelines

**Important**: After completing any task that modifies code, agents must rebuild the project to ensure changes are properly compiled:

```bash
go build -o hypr-exiled ./cmd/hypr-exiled
```

This ensures that any code changes are validated and the executable is updated with the latest modifications.

# Fixing Local vs Global Stats

Path of Exile 2 has two types of stats: local and global. The trade API uses different stat IDs for each type, and it's crucial to use the correct one based on the item category.

## Understanding Local vs Global Stats

- **Local stats**: Only affect the item they're on (e.g., "+# to Armour (Local)" on armor pieces)
- **Global stats**: Affect your character's total stats regardless of which item provides them (e.g., "+# to Armour" on belts/jewelry)

## When to Use Each Type

### Use Local Stats For:
- **Armor pieces**: helmets, gloves, boots, body armours, shields
- **Weapons**: when the stat affects the weapon's own properties
- Item categories starting with `armour.` in the trade API

### Use Global Stats For:
- **Jewelry**: rings, amulets, belts
- **Other accessories**: any item where the stat provides a global bonus
- Item categories starting with `accessory.` in the trade API

## Implementation Steps

When you discover a stat that needs local/global differentiation:

1. **Find the External Mapping**: Look up the stat in Exiled-Exchange-2's `stats.ndjson` file:
   ```bash
   grep -F '"# to <stat_name>"' /path/to/Exiled-Exchange-2/renderer/public/data/en/stats.ndjson
   ```

2. **Identify the Stat IDs**: The output will show multiple explicit IDs, typically:
   - First ID: Local version (for armor pieces)  
   - Second ID: Global version (for jewelry/accessories)

3. **Add Contextual Fix**: In both `buildStatFilters` and `buildPriceStatFilters` functions in `internal/input/input.go`, add a conditional mapping:

   ```go
   // Contextual fix for <stat_name> stats: use global on jewelry/belts, local on armor pieces
   if filter.ID == "explicit.stat_XXXXXXXX" { // local version ID
       if strings.HasPrefix(category, "accessory.") {
           // Use global stat for belts, rings, amulets
           filter.ID = "explicit.stat_YYYYYYYY" // global version ID
           i.log.Debug("Adjusted stat to global <stat_name> for accessory", "from", "explicit.stat_XXXXXXXX", "to", filter.ID, "text", stat.Text)
       }
       // Keep local version for armor pieces (armour.* categories)
   }
   ```

4. **Test the Fix**: 
   - Build the project: `go build -o hypr-exiled ./cmd/hypr-exiled`
   - Test with items from both categories (armor piece and jewelry)
   - Verify the correct stat IDs are used in the debug logs

5. **Add Debug Logging**: Include descriptive debug messages to track when the contextual fix is applied

## Example: Armor Stat Fix

The armor stat fix implemented in the codebase demonstrates this pattern:

```go
// Contextual fix for armor stats: use global on jewelry/belts, local on armor pieces
if filter.ID == "explicit.stat_3484657501" { // "# to Armour" (local version)
    if strings.HasPrefix(category, "accessory.") {
        // Use global armor stat for belts, rings, amulets
        filter.ID = "explicit.stat_809229260"
        i.log.Debug("Adjusted stat to global armor for accessory", "from", "explicit.stat_3484657501", "to", filter.ID, "text", stat.Text)
    }
    // Keep local version for armor pieces (armour.* categories)
}
```

This ensures belts use the global "+# to Armour" stat while body armours use the local "+# to Armour (Local)" stat.