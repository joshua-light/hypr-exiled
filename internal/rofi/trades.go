package rofi

import (
	"fmt"
	"hypr-exiled/internal/models"
	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/logger"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config holds the configuration for the Rofi menu.
type Config struct {
	Args    []string
	Message string
}

// ActionHandler defines a function type for handling custom actions.
type ActionHandler func(selected string) error

var (
	baseArgs = []string{
		"-dmenu",
		"-markup-rows",
		"-kb-custom-1", "t",
		"-show-icons",
		"-kb-custom-2", "p",
		"-kb-custom-3", "f",
		"-kb-custom-4", "d",
		"-kb-accept-entry", "Return",
		"-markup",
		"-eh", "2",
	}

	TradeConfig = Config{
		Args:    []string{}, // Will be populated in NewTradeDisplayManager
		Message: "T (trade) | P (party) | F (finish) | D (delete)",
	}
)

// TradeDisplayManager handles displaying content using Rofi.
type TradeDisplayManager struct {
	config        Config
	tradeHandler  ActionHandler
	partyHandler  ActionHandler
	finishHandler ActionHandler
	deleteHandler ActionHandler
	log           *logger.Logger
}

// NewDisplayManager creates a new DisplayManager instance.
func NewTradeDisplayManager(tradeHandler, partyHandler, finishHandler, deleteHandler ActionHandler) *TradeDisplayManager {
	log := global.GetLogger()
	log.Info("Initializing Rofi Trade DisplayManager")

	themePath, err := global.GetConfig().GetRofiThemePath()
	if err != nil {
		log.Error("Failed to get Rofi theme path", err)
	}

	args := append(baseArgs, "-theme", themePath)
	config := Config{
		Args:    args,
		Message: "T (trade) | P (party) | F (finish) | D (delete)",
	}

	return &TradeDisplayManager{
		config:        config,
		tradeHandler:  tradeHandler,
		partyHandler:  partyHandler,
		finishHandler: finishHandler,
		deleteHandler: deleteHandler,
		log:           log,
	}
}

func (d *TradeDisplayManager) FormatTrade(trade models.TradeEntry) string {
	config := global.GetConfig()

	currencySymbols := map[string]string{
		"divine":  fmt.Sprintf("\x00icon\x1f%s", filepath.Join(config.AssetsDir, "divine.png")),
		"exalted": fmt.Sprintf("\x00icon\x1f%s", filepath.Join(config.AssetsDir, "exalt.png")),
	}

	currencyStr := fmt.Sprintf("%.0f", trade.CurrencyAmount)
	if trade.CurrencyAmount != float64(int(trade.CurrencyAmount)) {
		currencyStr = fmt.Sprintf("%.2f", trade.CurrencyAmount)
	}

	currencyName := "Divs"
	if trade.CurrencyType == "exalted" {
		currencyName = "Exs"
	}

	symbol, exists := currencySymbols[trade.CurrencyType]
	if !exists {
		symbol = trade.CurrencyType
	}

	switch trade.TriggerType {
	case "incoming_trade":
		return fmt.Sprintf("%s %s > %s&#x0a;@%s%s",
			currencyStr,
			currencyName,
			trade.ItemName,
			trade.PlayerName,
			symbol)
	default: // outgoing_trade
		return fmt.Sprintf("%s %s > %s&#x0a;@%s%s",
			currencyStr,
			currencyName,
			trade.ItemName,
			trade.PlayerName,
			symbol)
	}
}

// ExtractPlayerName extracts the player name from the selected Rofi string.
func (d *TradeDisplayManager) ExtractPlayerName(selected string) (string, error) {
	parts := strings.Split(selected, "(@")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid selected string format: %s", selected)
	}

	playerPart := strings.Split(parts[1], ")")
	if len(playerPart) < 1 {
		return "", fmt.Errorf("invalid selected string format: %s", selected)
	}

	return playerPart[0], nil
}

// DisplayTrades displays the trades in a Rofi menu.
func (d *TradeDisplayManager) DisplayTrades(trades []string) error {
	d.log.Debug("Starting DisplayTrades", "trade_count", len(trades))
	if len(trades) == 0 {
		d.log.Warn("No trades to display")
		return fmt.Errorf("no trades to display")
	}

	args := append(d.config.Args, "-mesg", d.config.Message)
	d.log.Debug("Constructed Rofi command", "args", args)

	cmd := exec.Command("rofi", args...)
	cmd.Stdin = strings.NewReader(strings.Join(trades, "\n"))
	d.log.Info("Executing Rofi command", "command", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			d.log.Debug("Rofi exited with code", "exit_code", exitErr.ExitCode())
			return d.handleExitCode(string(output), exitErr.ExitCode())
		}
		d.log.Error("Failed to run Rofi", err)
		return fmt.Errorf("failed to run rofi: %w", err)
	}
	d.log.Debug("Rofi output", "output", string(output))
	return nil
}

// handleExitCode processes the Rofi exit code and executes the corresponding handler.
func (d *TradeDisplayManager) handleExitCode(selected string, exitCode int) error {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		d.log.Debug("No selection made in Rofi")
		return nil
	}
	d.log.Debug("Processing Rofi exit code", "exit_code", exitCode, "selected", selected)
	switch exitCode {
	case 10: // T pressed - Trade
		if d.tradeHandler != nil {
			d.log.Info("Trade action triggered", "selected", selected)
			return d.tradeHandler(selected)
		}
	case 11: // P pressed - Party
		if d.partyHandler != nil {
			d.log.Info("Party action triggered", "selected", selected)
			return d.partyHandler(selected)
		}
	case 12: // F pressed - Finish
		if d.finishHandler != nil {
			d.log.Info("Finish action triggered", "selected", selected)
			return d.finishHandler(selected)
		}
	case 13: // D pressed - Delete
		if d.deleteHandler != nil {
			d.log.Info("Delete action triggered", "selected", selected)
			return d.deleteHandler(selected)
		}
	}
	d.log.Warn("Unhandled Rofi exit code", "exit_code", exitCode)
	return nil
}
