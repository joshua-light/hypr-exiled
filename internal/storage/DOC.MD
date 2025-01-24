# Storage Package

## Database Structure

```sql
CREATE TABLE trades (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    trigger_type TEXT NOT NULL,
    player_name TEXT NOT NULL,
    item_name TEXT NOT NULL,
    league TEXT NOT NULL,
    currency_amount REAL NOT NULL,
    currency_type TEXT NOT NULL,
    stash_tab TEXT NOT NULL,
    position_left INTEGER NOT NULL,
    position_top INTEGER NOT NULL,
    message TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Key Operations

```go
// Initialize database
New() (*DB, error)

// Core operations
AddTrade(trade models.TradeEntry) error
GetTrades() ([]models.TradeEntry, error)
RemoveTradesByPlayer(playerName string) error
Cleanup(olderThan time.Duration) error
```

## Features

- WAL mode for concurrent access
- Auto-cleanup of old trades
- Transaction support
- Config directory integration

## Storage Location
```go
filepath.Join(os.UserConfigDir(), "hypr-exiled", "trades.db")
```

## Best Practices

- Transaction handling for batch operations
- Proper resource cleanup
- Error context preservation
- Timestamp-based cleanup
- Structured logging
