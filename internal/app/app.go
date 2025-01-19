package app

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"poe-helper/internal/models"
	"poe-helper/internal/poe/log"
	"poe-helper/pkg/global"
	"poe-helper/pkg/notify"
)

type POEHelper struct {
	entries       []models.TradeEntry
	poeLogWatcher *poe_log.LogWatcher
}

func NewPOEHelper() (*POEHelper, error) {
	log := global.GetLogger()
	config := global.GetConfig()

	log.Info("Creating new POE Helper instance")
	log.Debug("Initializing POEHelper",
		"log_path", config.PoeLogPath,
		"notify_command", config.NotifyCommand,
		"trigger_count", len(config.CompiledTriggers))

	if err := checkDependencies(); err != nil {
		log.Error("Dependency check failed", err,
			"details", "Required dependencies not found")
		global.GetNotifier().Show(err.Error(), notify.Error)
		return nil, err
	}

	helper := &POEHelper{
		entries: make([]models.TradeEntry, 0),
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
	log.Info("POEHelper initialized successfully",
		"watcher_status", "ready",
		"entry_buffer_size", len(helper.entries))
	return helper, nil
}

func checkDependencies() error {
	log := global.GetLogger()

	log.Info("Checking system dependencies")
	if _, err := exec.LookPath("rofi"); err != nil {
		log.Info("Dependency check failed",
			"missing_dependency", "rofi",
			"error", err)
		return fmt.Errorf("rofi is not installed. Please install it using your package manager")
	}
	log.Info("All dependencies satisfied")
	return nil
}

func (p *POEHelper) handleTradeEntry(entry models.TradeEntry) {
	notifier := global.GetNotifier()
	log := global.GetLogger()

	log.Info("Trade request received",
		"player", entry.PlayerName,
		"type", entry.TriggerType,
		"timestamp", entry.Timestamp,
		"entry_count", len(p.entries))

	log.Debug("Processing trade entry",
		"current_entries", len(p.entries),
		"new_entry_player", entry.PlayerName)
	p.entries = append(p.entries, entry)

	message := fmt.Sprintf("Trade request from %s", entry.PlayerName)
	log.Debug("Preparing notification",
		"message", message,
		"type", "info")

	if err := notifier.Show(message, notify.Info); err != nil {
		log.Error("Notification failed",
			err,
			"player", entry.PlayerName,
			"message", message)
	}
}

func (p *POEHelper) Run() error {
	notifier := global.GetNotifier()
	log := global.GetLogger()

	log.Info("Starting POE Helper service")
	log.Debug("Initializing service components")

	if err := notifier.Show("POE Helper started", notify.Info); err != nil {
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

func (p *POEHelper) Stop() error {
	log := global.GetLogger()

	log.Info("Initiating POE Helper shutdown")

	if p.poeLogWatcher != nil {
		log.Debug("Stopping log watcher")
		p.poeLogWatcher.Stop()
	}

	log.Info("POE Helper shutdown complete",
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
