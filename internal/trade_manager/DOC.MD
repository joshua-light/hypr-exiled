# Trade Manager Package

### Types (`types.go`)

```go
type TradeManager struct {
    db       *storage.DB
    rofi     *rofi.TradeDisplayManager
    input    *input.Input
    detector *window.Detector
    // ... other fields
}

type Trade struct {
    ID         string
    IsSell     bool
    Item       string
    Price      []Currency
    PlayerName string
}
```

## Key Features

### Trade Management
- Add new trades to database
- Display trades via Rofi UI
- Handle trade actions:
  - Initiate trade
  - Send party invites
  - Complete trades
  - Delete trades

### Automation Features
- Auto-cleanup of old trades (24h)
- Command templating with {player}
- Notification system integration
- Window state monitoring

## Primary Operations

### Trade Commands
```go
// Add new trade
AddTrade(trade models.TradeEntry) error

// Display trade UI
ShowTrades() error

// Handle trade actions
handleTrade(selected string) error
handleParty(selected string) error
handleFinish(selected string) error
handleDelete(selected string) error
```

### Implementation Details

#### Creating Manager
```go
tm := NewTradeManager(detector)
// Initializes:
// - Database connection
// - Input manager
// - Rofi UI manager
// - Trade handlers
```

#### Trade Actions
1. **Add Trade**:
   - Store in database
   - Send notification
   - Log trade details

2. **Show Trades**:
   - Verify PoE window
   - Fetch from database
   - Format for Rofi
   - Display UI

3. **Handle Trade**:
   - Extract player name
   - Process commands
   - Execute in-game actions

4. **Cleanup**:
   - Remove completed trades
   - Clean old entries
   - Close connections

### Best Practices

- **Database Operations**:
  - Proper transaction handling
  - Error checking
  - Cleanup routines

- **Command Execution**:
  - Template validation
  - Safe player name handling
  - Command sequence timing

- **Error Handling**:
  - Detailed error context
  - User notifications
  - Graceful degradation

- **Resource Management**:
  - Proper cleanup
  - Connection handling
  - Memory management
