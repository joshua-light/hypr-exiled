package app

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"poe-helper/internal/models"
	"poe-helper/internal/notify"
	"poe-helper/internal/poe/log"
	"poe-helper/pkg/logger"
	"syscall"
)

type POEHelper struct {
	config     *Config
	log        *logger.Logger
	entries    []models.TradeEntry
	notifier   *notify.NotifyService
	logWatcher *poe_log.LogWatcher
}

func NewPOEHelper(config *Config, log *logger.Logger) (*POEHelper, error) {
	log.Debug("Initializing POEHelper",
		"log_path", config.PoeLogPath,
		"notify_command", config.NotifyCommand)

	if err := checkDependencies(); err != nil {
		log.Error("Dependency check failed", err)
		tempNotifier := notify.NewNotifyService("", log)
		tempNotifier.Show(err.Error(), notify.Error)
		return nil, err
	}

	helper := &POEHelper{
		config:   config,
		log:      log,
		entries:  make([]models.TradeEntry, 0),
		notifier: notify.NewNotifyService(config.NotifyCommand, log),
	}

	log.Debug("Initializing log watcher",
		"trigger_count", len(config.CompiledTriggers))

	logWatcher, err := poe_log.NewLogWatcher(
		config.PoeLogPath,
		log,
		config.CompiledTriggers,
		helper.handleTradeEntry,
	)
	if err != nil {
		log.Error("Failed to initialize log watcher", err)
		return nil, fmt.Errorf("failed to initialize log watcher: %w", err)
	}

	helper.logWatcher = logWatcher
	log.Debug("POEHelper initialized successfully")
	return helper, nil
}

func checkDependencies() error {
	if _, err := exec.LookPath("rofi"); err != nil {
		return fmt.Errorf("rofi is not installed. Please install it using your package manager")
	}
	return nil
}

func (p *POEHelper) handleTradeEntry(entry models.TradeEntry) {
	p.log.Info("Trade request received",
		"player", entry.PlayerName,
		"type", entry.TriggerType,
		"timestamp", entry.Timestamp,
	)

	p.log.Debug("Adding entry to history",
		"current_count", len(p.entries))
	p.entries = append(p.entries, entry)

	message := fmt.Sprintf("Trade request from %s", entry.PlayerName)
	p.log.Debug("Showing notification", "message", message)

	if err := p.notifier.Show(message, notify.Info); err != nil {
		p.log.Error("Failed to show trade notification",
			err,
			"player", entry.PlayerName)
	}
}

func (p *POEHelper) Run() error {
	p.log.Debug("Starting POEHelper")

	if err := p.notifier.Show("POE Helper started. Monitoring for trade messages...", notify.Info); err != nil {
		p.log.Error("Failed to show startup notification", err)
	}

	go func() {
		p.log.Debug("Starting log watcher routine")
		if err := p.logWatcher.Watch(); err != nil {
			p.log.Error("Log watcher error", err)
			p.notifier.Show(fmt.Sprintf("Log watcher error: %v", err), notify.Error)
		}
	}()

	p.log.Debug("Waiting for shutdown signal")
	waitForShutdown()
	return p.Stop()
}

func (p *POEHelper) Stop() error {
	p.log.Debug("Stopping POEHelper")
	if p.logWatcher != nil {
		p.logWatcher.Stop()
	}
	return nil
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
