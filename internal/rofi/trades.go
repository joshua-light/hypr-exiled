package rofi

import (
	"fmt"
	"os/exec"
	"strings"

	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/logger"
)

// Config holds the configuration for the Rofi menu.
type Config struct {
	Args    []string
	Message string
}

// ActionHandler defines a function type for handling custom actions.
type ActionHandler func(selected string) error

var (
	TradeConfig = Config{
		Args: []string{
			"-dmenu",
			"-markup-rows",
			"-kb-custom-1", "t",
			"-kb-custom-2", "p",
			"-kb-custom-3", "f",
			"-kb-custom-4", "d",
			"-kb-accept-entry", "Return",
			"-theme", "~/.config/rofi/trade.rasi",
		},
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

	return &TradeDisplayManager{
		config:        TradeConfig,
		tradeHandler:  tradeHandler,
		partyHandler:  partyHandler,
		finishHandler: finishHandler,
		deleteHandler: deleteHandler,
		log:           log,
	}
}

// DisplayTrades displays the trades in a Rofi menu.
func (d *TradeDisplayManager) DisplayTrades(trades []string) error {
	d.log.Debug("Starting DisplayTrades", "trade_count", len(trades))

	if len(trades) == 0 {
		d.log.Warn("No trades to display")
		return fmt.Errorf("no trades to display")
	}

	// Create Rofi command with -mesg
	args := append(d.config.Args, "-mesg", d.config.Message)
	d.log.Debug("Constructed Rofi command", "args", args)

	// Create Rofi command
	cmd := exec.Command("rofi", args...)
	cmd.Stdin = strings.NewReader(strings.Join(trades, "\n"))

	d.log.Info("Executing Rofi command", "command", cmd.String())

	// Run Rofi and capture output
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
