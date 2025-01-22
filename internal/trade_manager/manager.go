package trade_manager

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"poe-helper/internal/models"
	"poe-helper/pkg/global"
	"poe-helper/pkg/logger"
)

var defaultRofiConfig = RofiConfig{
	Args: []string{
		"-dmenu",
		"-markup-rows",
		"-multi-select",
		"-kb-custom-1", "t",
		"-kb-custom-2", "p",
		"-kb-custom-3", "f",
		"-kb-custom-4", "d",
		"-kb-accept-entry", "Return",
		"-theme", "~/.config/rofi/trade.rasi",
	},
	Message: "T (trade) | P (party) | F (finish) | D (delete)",
}

type Manager struct {
	trades []models.TradeEntry
	mu     sync.RWMutex
	log    *logger.Logger
}

func NewManager() *Manager {
	return &Manager{
		trades: make([]models.TradeEntry, 0),
		log:    global.GetLogger(),
	}
}

// AddTrade only adds the trade to the list without showing Rofi
func (m *Manager) AddTrade(trade models.TradeEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.Debug("Adding new trade",
		"player", trade.PlayerName,
		"item", trade.ItemName,
		"trigger", trade.TriggerType)

	// Check for existing trade with same characteristics
	exists := false
	for i, existing := range m.trades {
		if existing.PlayerName == trade.PlayerName &&
			existing.ItemName == trade.ItemName &&
			existing.Position.Left == trade.Position.Left &&
			existing.Position.Top == trade.Position.Top {
			m.trades[i] = trade // Update existing entry
			exists = true
			break
		}
	}

	if !exists {
		m.trades = append(m.trades, trade)
	}

	return nil
}

func (m *Manager) ShowTrades() error {
	selected, exitCode, err := m.showRofi()
	if err != nil {
		return fmt.Errorf("rofi display error: %w", err)
	}

	m.handleTrades(selected, exitCode)
	return nil
}

// showRofi displays the trade list and returns user selection
func (m *Manager) showRofi() (string, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.trades) == 0 {
		m.log.Debug("No trades to display")
		return "", 0, nil
	}

	var options []string
	for _, trade := range m.trades {
		options = append(options, formatTrade(trade))
	}

	m.log.Debug("Showing rofi window",
		"trade_count", len(options))

	cmd := exec.Command("rofi", defaultRofiConfig.Args...)
	cmd.Stdin = strings.NewReader(strings.Join(options, "\n"))

	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), exitErr.ExitCode(), nil
		}
		return "", 0, err
	}

	return string(output), 0, nil
}

func (m *Manager) removeTrades(toRemove []models.TradeEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newTrades := make([]models.TradeEntry, 0, len(m.trades))
	for _, trade := range m.trades {
		shouldKeep := true
		for _, remove := range toRemove {
			if trade.PlayerName == remove.PlayerName &&
				trade.ItemName == remove.ItemName &&
				trade.Position.Left == remove.Position.Left &&
				trade.Position.Top == remove.Position.Top {
				shouldKeep = false
				break
			}
		}
		if shouldKeep {
			newTrades = append(newTrades, trade)
		}
	}
	m.trades = newTrades
}

func (m *Manager) handleTrades(selected string, exitCode int) {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		return
	}

	var selectedTrades []models.TradeEntry
	m.mu.RLock()
	for _, trade := range m.trades {
		formattedTrade := formatTrade(trade)
		if strings.Contains(selected, formattedTrade) {
			selectedTrades = append(selectedTrades, trade)
		}
	}
	m.mu.RUnlock()

	m.log.Debug("Processing trade action",
		"exit_code", exitCode,
		"selected_trades", len(selectedTrades))

	switch exitCode {
	case 10: // T pressed - Trade
		for _, trade := range selectedTrades {
			m.log.Info("Initiating trade",
				"player", trade.PlayerName,
				"item", trade.ItemName)
			// TODO: Implement trade command execution
		}
		m.ShowTrades() // Show trades again after action
	case 11: // P pressed - Party
		for _, trade := range selectedTrades {
			m.log.Info("Sending party invite",
				"player", trade.PlayerName)
			// TODO: Implement party invite command execution
		}
		m.ShowTrades() // Show trades again after action
	case 12: // F pressed - Finish
		m.log.Info("Finishing trades", "count", len(selectedTrades))
		m.removeTrades(selectedTrades)
		m.ShowTrades() // Show remaining trades if any
	case 13: // D pressed - Delete
		m.log.Info("Deleting trades", "count", len(selectedTrades))
		m.removeTrades(selectedTrades)
		m.ShowTrades() // Show remaining trades if any
	}
}

// Helper function to format trade entries for display
func formatTrade(trade models.TradeEntry) string {
	currencyStr := fmt.Sprintf("%.0f", trade.CurrencyAmount)
	if trade.CurrencyAmount != float64(int(trade.CurrencyAmount)) {
		currencyStr = fmt.Sprintf("%.2f", trade.CurrencyAmount)
	}

	priceStr := fmt.Sprintf("%s %s", currencyStr, trade.CurrencyType)

	switch trade.TriggerType {
	case "incoming_trade":
		return fmt.Sprintf("%s > %s <span size='small'>(@%s)</span>",
			trade.ItemName,
			priceStr,
			trade.PlayerName)
	default: // outgoing_trade
		return fmt.Sprintf("%s > %s <span size='small'>(@%s)</span>",
			priceStr,
			trade.ItemName,
			trade.PlayerName)
	}
}
