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
- **`getDefaultPoeLogPath` Function**: Finds the default Path of Exile log file by checking common locations.

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
