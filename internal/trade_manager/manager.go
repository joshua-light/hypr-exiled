package trade_manager

import (
	"fmt"
	"strings"
	"time"

	"hypr-exiled/internal/input"
	"hypr-exiled/internal/models"
	"hypr-exiled/internal/poe/window"
	"hypr-exiled/internal/rofi"
	"hypr-exiled/internal/storage"
	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/notify"
)

func NewTradeManager(detector *window.Detector, input *input.Input) *TradeManager {
	cfg, log, notifier := global.GetAll()

	db, err := storage.New()
	if err != nil {
		log.Fatal("Failed to initialize storage", err)
	}

	// Cleanup old trades (older than 24 hours)
	go func() {
		if err := db.Cleanup(24 * time.Hour); err != nil {
			log.Error("Failed to cleanup old trades", err)
		}
	}()

	// Create the TradeManager instance
	tm := &TradeManager{
		db:       db,
		notify:   notifier,
		input:    input,
		detector: detector,
		cfg:      cfg,
		log:      log,
	}

	// Initialize Rofi with handlers that have access to the TradeManager instance
	rofiManager := rofi.NewTradeDisplayManager(
		func(selected string) error { return tm.handleTrade(selected) },
		func(selected string) error { return tm.handleParty(selected) },
		func(selected string) error { return tm.handleFinish(selected) },
		func(selected string) error { return tm.handleDelete(selected) },
	)

	tm.rofi = rofiManager
	return tm
}

func (tm *TradeManager) AddTrade(trade models.TradeEntry) error {
	tm.log.Debug("Adding trade", "trade", trade)
	if err := tm.db.AddTrade(trade); err != nil {
		tm.log.Error("Failed to add trade", err)
		return fmt.Errorf("failed to add trade: %w", err)
	}

	var notificationMsg string
	if trade.TriggerType == "incoming_trade" {
		notificationMsg = fmt.Sprintf("@%s wants to buy %s for %.0f %s",
			trade.PlayerName,
			trade.ItemName,
			trade.CurrencyAmount,
			trade.CurrencyType)

		// Play notification sound for incoming trades
		if notifier := global.GetSoundNotifier(); notifier != nil {
			if err := notifier.PlayTradeSound(); err != nil {
				tm.log.Error("Failed to play trade sound", err)
			}
		}
	} else {
		notificationMsg = fmt.Sprintf("Trade request for %s sent to @%s",
			trade.ItemName,
			trade.PlayerName,
		)
	}

	if err := global.GetNotifier().Show(notificationMsg, notify.Info); err != nil {
		tm.log.Error("Failed to send trade notification", err)
	}

	tm.log.Info("Trade added successfully", "trade", trade)
	return nil
}

func (tm *TradeManager) ShowTrades() error {
	if !tm.detector.IsActive() {
		tm.notify.Show("PoE 2 Window not found, make sure PoE is open", notify.Info)
		tm.log.Debug("PoE 2 window not found")
		return nil
	}
	tm.log.Debug("Fetching trades from database")
	trades, err := tm.db.GetTrades()
	if err != nil {
		tm.log.Error("Failed to get trades", err)
		return fmt.Errorf("failed to get trades: %w", err)
	}

	if len(trades) == 0 {
		tm.notify.Show("No trades to display", notify.Info)
		tm.log.Debug("No trades to display")
		return nil
	}

	// Format trades for Rofi
	var options []string
	for i, trade := range trades {
		formattedTrade := tm.rofi.FormatTrade(trade, i)
		options = append(options, formattedTrade)
		tm.log.Debug("Adding trade to options",
			"index", i,
			"player_name", trade.PlayerName)
	}

	tm.log.Info("Displaying trades in Rofi", "trade_count", len(trades))
	if err := tm.rofi.DisplayTrades(options); err != nil {
		tm.log.Error("Failed to display trades in Rofi", err)
		return fmt.Errorf("failed to show trades in rofi: %w", err)
	}

	return nil
}

func (tm *TradeManager) handleTrade(selected string) error {
	playerName, err := tm.rofi.ExtractPlayerName(selected)
	if err != nil {
		return fmt.Errorf("failed to extract player name: %w", err)
	}

	commands := tm.cfg.GetCommands()["trade"]
	for i := range commands {
		commands[i] = strings.ReplaceAll(commands[i], "{player}", playerName)
	}

	if err := tm.input.ExecutePoECommands(commands); err != nil {
		return fmt.Errorf("failed to execute trade commands: %w", err)
	}

	return nil
}

func (tm *TradeManager) handleParty(selected string) error {
	tm.log.Debug("Handling party request", "selected_trade", selected)

	playerName, err := tm.rofi.ExtractPlayerName(selected)
	if err != nil {
		tm.log.Error("Failed to extract player name", err)
		return fmt.Errorf("failed to extract player name: %w", err)
	}

	tm.log.Debug("Extracted player name for party", "player_name", playerName)

	commands := tm.cfg.GetCommands()["party"]
	tm.log.Debug("Original commands", "commands", commands) // Log original commands

	for i := range commands {
		originalCmd := commands[i]
		commands[i] = strings.ReplaceAll(commands[i], "{player}", playerName)

		tm.log.Debug("Preparing party command",
			"original_command", originalCmd,
			"modified_command", commands[i])
	}

	tm.log.Debug("Modified commands", "commands", commands) // Log modified commands

	if err := tm.input.ExecutePoECommands(commands); err != nil {
		tm.log.Error("Failed to execute party commands", err)
		return fmt.Errorf("failed to execute party commands: %w", err)
	}

	return nil
}

func (tm *TradeManager) handleFinish(selected string) error {
	playerName, err := tm.rofi.ExtractPlayerName(selected)
	if err != nil {
		return fmt.Errorf("failed to extract player name: %w", err)
	}

	commands := tm.cfg.GetCommands()["finish"]
	for i := range commands {
		commands[i] = strings.ReplaceAll(commands[i], "{player}", playerName)
	}

	if err := tm.input.ExecutePoECommands(commands); err != nil {
		return fmt.Errorf("failed to execute finish commands: %w", err)
	}

	// Remove trade from database
	if err := tm.db.RemoveTradesByPlayer(playerName); err != nil {
		return fmt.Errorf("failed to remove trades: %w", err)
	}

	return nil
}

func (tm *TradeManager) handleDelete(selected string) error {
	tm.log.Info("Delete action triggered", "selected", selected)

	playerName, err := tm.rofi.ExtractPlayerName(selected)
	if err != nil {
		tm.log.Error("Failed to extract player name", err, "selected", selected)
		return fmt.Errorf("failed to extract player name: %w", err)
	}

	if err := tm.db.RemoveTradesByPlayer(playerName); err != nil {
		tm.log.Error("Failed to delete trade", err, "player_name", playerName)
		return fmt.Errorf("failed to delete trade: %w", err)
	}

	tm.ShowTrades()

	tm.log.Info("Trade deleted from the database", "player_name", playerName)
	return nil
}

func (m *TradeManager) Close() error {
	m.log.Info("Closing TradeManager")
	return m.db.Close()
}
