package app

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"hypr-exiled/internal/models"
	poe_log "hypr-exiled/internal/poe/log"
	"hypr-exiled/internal/trade_manager"
	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/notify"
)

type HyprExiled struct {
	entries       []models.TradeEntry
	poeLogWatcher *poe_log.LogWatcher
	TradeManager  *trade_manager.TradeManager
}

func NewHyprExiled() (*HyprExiled, error) {
	log := global.GetLogger()
	config := global.GetConfig()

	log.Info("Creating new Hypr Exiled instance")
	log.Debug("Initializing HyprExiled",
		"log_path", config.PoeLogPath,
		"notify_command", config.NotifyCommand,
		"trigger_count", len(config.CompiledTriggers))

	if err := checkDependencies(); err != nil {
		log.Error("Dependency check failed", err,
			"details", "Required dependencies not found")
		global.GetNotifier().Show(err.Error(), notify.Error)
		return nil, err
	}

	tradeManager := trade_manager.NewTradeManager()

	helper := &HyprExiled{
		entries:      make([]models.TradeEntry, 0),
		TradeManager: tradeManager, // Use TradeManager (uppercase T)
	}

	log.Debug("Creating log watcher instance")
	logWatcher, err := poe_log.NewLogWatcher(
		helper.handleTradeEntry,
	)

	if err != nil {
		log.Error("Log watcher initialization failed",
			err,
			"details", "Failed to create log watcher instance")
		return nil, fmt.Errorf("failed to initialize log watcher: %w", err)
	}

	helper.poeLogWatcher = logWatcher
	return helper, nil
}

func checkDependencies() error {
	log := global.GetLogger()

	log.Info("Checking system dependencies")
	deps := []string{"rofi"}
	for _, dep := range deps {
		if _, err := exec.LookPath(dep); err != nil {
			log.Info("Dependency check failed",
				"missing_dependency", dep,
				"error", err)
			return fmt.Errorf("%s is not installed. Please install it using your package manager", dep)
		}
	}
	log.Info("All dependencies satisfied")
	return nil
}

func (p *HyprExiled) Run() error {
	notifier := global.GetNotifier()
	log := global.GetLogger()

	log.Info("Starting Hypr Exiled service")
	log.Debug("Initializing service components")

	if err := notifier.Show("Hypr Exiled started", notify.Info); err != nil {
		log.Error("Startup notification failed",
			err,
			"notification_type", "startup")
	}

	go func() {
		if err := p.poeLogWatcher.Watch(); err != nil {
			log.Error("Log watcher routine failed",
				err,
				"component", "log_watcher")
			notifier.Show(fmt.Sprintf("Log watcher error: %v", err), notify.Error)
		}
	}()

	log.Info("Service started successfully",
		"status", "running",
		"waiting_for", "shutdown_signal")
	waitForShutdown()
	return p.Stop()
}

func (p *HyprExiled) Stop() error {
	log := global.GetLogger()

	log.Info("Initiating Hypr Exiled shutdown")

	if p.poeLogWatcher != nil {
		log.Debug("Stopping log watcher")
		p.poeLogWatcher.Stop()
	}

	log.Info("Hypr Exiled shutdown complete",
		"status", "stopped",
		"processed_entries", len(p.entries))
	return nil
}

func waitForShutdown() {
	log := global.GetLogger()
	log.Debug("Setting up shutdown signal handler",
		"signals", []string{"SIGINT", "SIGTERM"})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Info("Shutdown signal received",
		"signal", sig.String())
}

func (p *HyprExiled) handleTradeEntry(entry models.TradeEntry) {
	log := global.GetLogger()

	if err := p.TradeManager.AddTrade(entry); err != nil {
		log.Error("Failed to process trade in manager",
			err,
			"player", entry.PlayerName,
			"item", entry.ItemName)
	}
}
