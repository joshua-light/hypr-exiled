 `pkg/config` Documentation

The `pkg/config` package is responsible for managing the application's configuration. It handles loading configuration from files, setting up default values, and providing access to configuration data in a controlled and immutable way.
---

## Table of Contents
1. [Overview](#overview)
2. [Structure](#structure)
3. [Files](#files)
   - [config.go](#configgo)
   - [defaults.go](#defaultsgo)
   - [file_loader.go](#file_loadergo)
   - [games.go](#gamesgo)
   - [logpaths.go](#logpathsgo)
   - [triggers.go](#triggersgo)
   - [commands.go](#commandsgo)
   - [assets.go](#assetsgo)
   - [utils.go](#utilsgo)
---

## Overview

The `pkg/config` package:
- Loads configuration from a JSON file or creates a default configuration if no file exists.
- Compiles regex patterns for triggers.
- Manages assets (e.g., icons, themes) used by the application.
- Provides getter methods to access configuration data in an immutable way.
- Maintains a configurable registry of supported Steam apps (name, AppID, window class) to enable PoE1/PoE2 switching without hardcoded values.
- **Log path resolution**: Selects the correct `client.txt` per game using clear precedence rules and optional per-AppID overrides.

---

## Structure

The package is organized into multiple files, each handling a specific aspect of configuration management:

- **`config.go`**: Defines the `Config` struct and its initialization.
- **`defaults.go`**: Handles the creation of default configuration values.
- **`file_loader.go`**: Loads configuration from a JSON file.
- **`triggers.go`**: Manages regex triggers and their compilation.
- **`commands.go`**: Provides access to command-related configuration.
- **`assets.go`**: Manages asset-related functionality (e.g., icons, themes).
- **`utils.go`**: Contains utility functions for configuration initialization and path resolution.
- **`games.go`**: Central Steam app registry (names, AppIDs, window classes) and lookup helpers.
- **`logpaths.go`**: Log path resolver for PoE1/PoE2 switching (per-AppID overrides and sibling path derivation).

---

## Files

### `config.go`
Defines the `Config` struct and its initialization.

#### Key Components:
- **`Config` Struct**: Holds all configuration data, including paths, triggers, commands, and assets.
- **`New` Function**: Creates a new `Config` instance with a provided logger.


---

### `defaults.go`
Handles the creation of default configuration values.

#### Key Components:
- **`DefaultConfig` Function**: Creates a default configuration with predefined values for triggers, commands, and paths.
- **`getDefaultPoeLogPath` Function** now delegates to `GetDefaultPoeLogPathFor(..., "Path of Exile 2")`.
- **`GetDefaultPoeLogPathFor` Function**: Finds the default Path of Exile log file by checking common locations given game folder name (`"Path of Exile"` or `"Path of Exile 2"`).  


---

### `file_loader.go`
Loads configuration from a JSON file.

#### Key Components:
- **`LoadFromFile` Method**: Reads and parses a JSON configuration file into the `Config` struct.
- **`loadConfigFromPath` Function**: Helper function to load configuration from a specific path.

---

### `triggers.go`
Manages regex triggers and their compilation.

#### Key Components:
- **`compile` Method**: Compiles regex patterns for triggers.
- **`GetTriggers` Method**: Returns a copy of the triggers map.
- **`GetCompiledTriggers` Method**: Returns a copy of the compiled triggers map.

---

### `commands.go`
Provides access to command-related configuration.

#### Key Components:
- **`GetCommands` Method**: Returns a copy of the commands map.
- **`GetNotifyCommand` Method**: Returns the notify command.

---

### `assets.go`
Manages asset-related functionality (e.g., icons, themes).

#### Key Components:
- **`GetAssetsDir` Method**: Returns the assets directory path.
- **`GetCurrencyIconPath` Method**: Returns the path to a currency icon.
- **`GetRofiThemePath` Method**: Returns the path to the Rofi theme.
- **`setupAssets` Method**: Sets up the assets directory and copies embedded assets.

---

### `utils.go`
Contains utility functions for configuration initialization and path resolution.

#### Key Components:
- **`initializeConfig` Function**: Creates or loads the configuration from a file or default values.
- **`FindConfig` Function**: Locates and initializes the configuration, setting up assets if necessary.

---

### `games.go`
Central registry and helpers for Steam apps.

#### Purpose
Defines the `SteamAppSpec` type and provides defaults and lookup helpers used by the window detector and the app layer to avoid hardcoded AppIDs or window classes.

#### Key Components:
- **`SteamAppSpec` Struct**: Holds names, appID and window classes for games.
- **Defaults**
  - Built-in fallback registry for PoE1/PoE2 if the user config omits `steam_apps`.
- **`GetSteamApps` Method**: Returns configured registry or defaults.
- **`GetDefaultAppID` Method**: Returns configured default AppID or PoE2 (2694490).
- **`GameNameByAppID` Method**: Maps AppID → human-readable name.
- **`WindowClasses` Method**: Collects configured window classes for detection.
- **`AppIDByWindowClass` Method**: Maps window class → AppID.

---

### `logpaths.go`
Log path resolver for multi-title support.

#### Purpose
Resolves the correct `client.txt` path **per AppID** with predictable precedence, enabling robust switching between PoE1 and PoE2 even on custom SteamLibrary locations.

#### Key Components:
- **`ResolveLogPathForAppID` Method** : Resolution precedence:
  1. **`log_paths` override**: use `log_paths["<appID>"]` if present and exists on disk.  
     If `log_paths` is present **but missing** the requested AppID, no guessing is done — an error is returned.
  2. **Sibling swap from `poe_log_path`**: if a single `poe_log_path` is configured, derive the other title’s path by swapping the game folder name (`"Path of Exile"` ⇄ `"Path of Exile 2"`).  
     - If the configured `poe_log_path` already matches the target game name and the file exists, use it directly.
     - If derivation fails (file not found or folder name not present), an error is returned with guidance to add `log_paths`.
  3. **Defaults**: Only when **neither** `log_paths` **nor** `poe_log_path` is provided, fall back to `GetDefaultPoeLogPathFor(log, gameName)`.

**Behavior notes**
- No silent fallback to the “other” game file when user config is present; errors are explicit to avoid misleading switches.
- JSON map keys for `log_paths` must be strings (e.g. `"238960"`), not integers.

---

### Example (JSON) config snippet
```json
{
  "default_app_id": 2694490,
  "steam_apps": [
    { "name": "Path of Exile",   "app_id": 238960,  "window_class": "steam_app_238960" },
    { "name": "Path of Exile 2", "app_id": 2694490, "window_class": "steam_app_2694490" }
  ],
  "log_paths": {
    "238960": "/mnt/data/SteamLibrary/steamapps/common/Path of Exile/logs/Client.txt",
    "2694490": "/mnt/data/SteamLibrary/steamapps/common/Path of Exile 2/logs/Client.txt"
  }
}