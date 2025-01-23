package trade_manager

import (
	"fmt"
	"path/filepath"
	"time"

	"hypr-exiled/internal/models"
	"hypr-exiled/internal/rofi"
	"hypr-exiled/internal/storage"
	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/notify"
)

// NewTradeManager creates a new TradeManager instance.
func NewTradeManager() *TradeManager {
	log := global.GetLogger()

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

	rofiManager := rofi.NewTradeDisplayManager(
		handleTrade,  // Trade handler
		handleParty,  // Party handler
		handleFinish, // Finish handler
		handleDelete, // Delete handler
	)

	return &TradeManager{
		db:   db,
		rofi: rofiManager,
		log:  log,
	}
}

// AddTrade adds a new trade to the database.
func (m *TradeManager) AddTrade(trade models.TradeEntry) error {
	m.log.Debug("Adding trade", "trade", trade)

	if err := m.db.AddTrade(trade); err != nil {
		m.log.Error("Failed to add trade", err)
		return fmt.Errorf("failed to add trade: %w", err)
	}

	// Create notification message based on trade type
	var notificationMsg string
	if trade.TriggerType == "incoming_trade" {
		notificationMsg = fmt.Sprintf("@%s wants to buy %s for %.0f %s",
			trade.PlayerName,
			trade.ItemName,
			trade.CurrencyAmount,
			trade.CurrencyType)
	} else {
		notificationMsg = fmt.Sprintf("Trade request for %s sent to @%s",
			trade.ItemName,
			trade.PlayerName,
		)
	}

	// Send notification
	if err := global.GetNotifier().Show(notificationMsg, notify.Info); err != nil {
		m.log.Error("Failed to send trade notification", err)
	}

	m.log.Info("Trade added successfully", "trade", trade)
	return nil
}

// ShowTrades displays the trades using Rofi.
func (m *TradeManager) ShowTrades() error {
	m.log.Debug("Fetching trades from database")
	trades, err := m.db.GetTrades()
	if err != nil {
		m.log.Error("Failed to get trades", err)
		return fmt.Errorf("failed to get trades: %w", err)
	}

	if len(trades) == 0 {
		m.log.Debug("No trades to display")
		return nil
	}

	// Format trades for Rofi
	var options []string
	for _, trade := range trades {
		formattedTrade := formatTrade(trade)
		options = append(options, formattedTrade)
		m.log.Debug("Formatted trade", "trade", formattedTrade)
	}

	m.log.Info("Displaying trades in Rofi", "trade_count", len(trades))
	if err := m.rofi.DisplayTrades(options); err != nil {
		m.log.Error("Failed to display trades in Rofi", err)
		return fmt.Errorf("failed to show trades in rofi: %w", err)
	}

	return nil
}

// handleTrade processes the trade action.
func handleTrade(selected string) error {
	log := global.GetLogger()
	log.Info("Trade action triggered", "selected", selected)
	// Implement trade logic here
	return nil
}

// handleParty processes the party action.
func handleParty(selected string) error {
	log := global.GetLogger()
	log.Info("Party action triggered", "selected", selected)
	// Implement party logic here
	return nil
}

// handleFinish processes the finish action.
func handleFinish(selected string) error {
	log := global.GetLogger()
	log.Info("Finish action triggered", "selected", selected)
	// Implement finish logic here
	return nil
}

// handleDelete processes the delete action.
func handleDelete(selected string) error {
	log := global.GetLogger()
	log.Info("Delete action triggered", "selected", selected)
	// Implement delete logic here
	return nil
}

func formatTrade(trade models.TradeEntry) string {
	config := global.GetConfig()

	currencySymbols := map[string]string{
		"divine":  fmt.Sprintf("\x00icon\x1f%s", filepath.Join(config.AssetsDir, "divine.png")),
		"exalted": fmt.Sprintf("\x00icon\x1f%s", filepath.Join(config.AssetsDir, "exalt.png")),
	}

	currencyStr := fmt.Sprintf("%.0f", trade.CurrencyAmount)
	if trade.CurrencyAmount != float64(int(trade.CurrencyAmount)) {
		currencyStr = fmt.Sprintf("%.2f", trade.CurrencyAmount)
	}

	symbol, exists := currencySymbols[trade.CurrencyType]
	if !exists {
		symbol = trade.CurrencyType
	}

	switch trade.TriggerType {
	case "incoming_trade":
		return fmt.Sprintf("%s > %s (@%s)%s",
			trade.ItemName,
			currencyStr,
			trade.PlayerName,
			symbol)
	default: // outgoing_trade
		return fmt.Sprintf("%s > %s (@%s)%s",
			currencyStr,
			trade.ItemName,
			trade.PlayerName,
			symbol)
	}
}

// Close closes the database connection.
func (m *TradeManager) Close() error {
	m.log.Info("Closing TradeManager")
	return m.db.Close()
}
