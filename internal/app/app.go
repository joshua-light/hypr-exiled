package app

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"hypr-exiled/internal/input"
	"hypr-exiled/internal/ipc"
	"hypr-exiled/internal/models"
	poe_log "hypr-exiled/internal/poe/log"
	"hypr-exiled/internal/poe/window"
	"hypr-exiled/internal/trade_manager"
	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/notify"
)

type HyprExiled struct {
	entries       []models.TradeEntry
	poeLogWatcher *poe_log.LogWatcher
	TradeManager  *trade_manager.TradeManager
	detector      *window.Detector
	input         *input.Input
}

func NewHyprExiled() (*HyprExiled, error) {
	log := global.GetLogger()
	config := global.GetConfig()

	log.Info("Creating new Hypr Exiled instance")
	log.Debug("Initializing HyprExiled",
		"log_path", config.GetPoeLogPath(),
		"notify_command", config.GetNotifyCommand(),
		"trigger_count", len(config.GetTriggers()))

	if err := checkDependencies(); err != nil {
		log.Error("Dependency check failed", err,
			"details", "Required dependencies not found")
		global.GetNotifier().Show(err.Error(), notify.Error)
		return nil, err
	}

	detector := window.NewDetector()
	if err := detector.Start(); err != nil {
		log.Error("Failed to start window detector", err)
		return nil, err
	}

	input, err := input.NewInput(detector)
	if err != nil {
		log.Fatal("Failed to initialize input handler", err)
	}

	tradeManager := trade_manager.NewTradeManager(detector, input)

	helper := &HyprExiled{
		entries:      make([]models.TradeEntry, 0),
		TradeManager: tradeManager,
		detector:     detector,
		input:        input,
	}

	logWatcher, err := poe_log.NewLogWatcher(
		helper.handleTradeEntry,
		detector,
	)
	if err != nil {
		log.Error("Log watcher initialization failed",
			err,
			"details", "Failed to create log watcher instance")
		return nil, fmt.Errorf("failed to initialize log watcher: %w", err)
	}

	initialAppID := detector.ActiveAppID()
	initialPath, err := config.ResolveLogPathForAppID(log, initialAppID)
	if err != nil {
		global.GetNotifier().Show(
			fmt.Sprintf("Log path resolution failed for %s: %v",
				config.GameNameByAppID(initialAppID), err),
			notify.Error)
		return nil, fmt.Errorf("log path resolution failed: %w", err)
	}

	log.Debug("Resolved initial log path", "app_id", initialAppID, "game", config.GameNameByAppID(initialAppID), "path", initialPath)
	logWatcher.SetPathOverride(initialPath)

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
	log.Info("Starting IPC socket server")
	go ipc.StartSocketServer(p.TradeManager, p.input)

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

	// react to AppID changes from Detector
	go p.handleAppIDChanges()

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

	if p.detector != nil {
		log.Debug("Stopping window detector")
		_ = p.detector.Stop()
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

func (p *HyprExiled) handleAppIDChanges() {
	log := global.GetLogger()
	notifier := global.GetNotifier()
	cfg := global.GetConfig()

	lastAppID := p.detector.ActiveAppID()

	for newAppID := range p.detector.Changes() {
		if newAppID == lastAppID {
			continue
		}

		gameName := cfg.GameNameByAppID(newAppID)
		newPath, err := cfg.ResolveLogPathForAppID(log, newAppID)
		if err != nil {
			log.Error("Failed to resolve log path for new game", err,
				"game", gameName,
				"app_id", newAppID)
			notifier.Show(fmt.Sprintf("Log path for %s not found. Set log_paths[%d] in config.", gameName, newAppID), notify.Error)
			continue
		}

		log.Info("Switching log watcher to new game",
			"from_app_id", lastAppID,
			"to_app_id", newAppID,
			"game", gameName,
			"path", newPath)

		// gracefully stop old Watcher
		if p.poeLogWatcher != nil {
			_ = p.poeLogWatcher.Stop()
		}

		// create & start new Watcher
		nw, err := poe_log.NewLogWatcher(p.handleTradeEntry, p.detector)
		if err != nil {
			log.Error("Failed to create new log watcher after app switch", err)
			continue
		}
		nw.SetPathOverride(newPath)
		p.poeLogWatcher = nw

		go func() {
			if err := p.poeLogWatcher.Watch(); err != nil {
				log.Error("Log watcher routine failed after app switch", err)
				notifier.Show(fmt.Sprintf("Log watcher error: %v", err), notify.Error)
			}
		}()

		notifier.Show(fmt.Sprintf("Switched to %s logs", cfg.GameNameByAppID(newAppID)), notify.Info)
		lastAppID = newAppID
	}
}
