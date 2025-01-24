# Rofi Trade Interface Manager

## Core Types

```go
type TradeDisplayManager struct {
    config        Config
    tradeHandler  ActionHandler
    partyHandler  ActionHandler
    finishHandler ActionHandler
    deleteHandler ActionHandler
}

type Config struct {
    Args    []string
    Message string
}
```

## Features

### Trade Display
- Custom Rofi theme integration
- Currency icon support (Divine/Exalted)
- Formatted trade listings
- Keyboard shortcuts

### Actions
```
T: Trade
P: Party
F: Finish
D: Delete
```

### Trade Formatting
- Index-based identification
- Currency amount formatting
- Player name extraction
- Icon integration

## Implementation

### Display Configuration
```go
baseArgs = []string{
    "-dmenu",
    "-markup-rows",
    "-show-icons",
    "-markup",
    "-eh", "2"
    // keyboard bindings
}
```

### Key Functions

#### Format Trade
```go
FormatTrade(trade models.TradeEntry, index int) string
// [0] 5 Divs > Item Name
// @PlayerName {icon}
```

#### Extract Player
```go
ExtractPlayerName(selected string) (string, error)
// Regex: \[\d+\] .*&#x0a;@([^\s]+)
```

#### Display UI
```go
DisplayTrades(trades []string) error
// Shows Rofi menu
// Handles action selection
```

### Exit Codes
- 10: Trade action
- 11: Party invite
- 12: Finish trade
- 13: Delete trade

## Best Practices

- Handle empty trade lists
- Validate player names
- Format currency amounts
- Clean exit handling
- Proper icon paths
