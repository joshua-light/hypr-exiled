package global

import (
	"poe-helper/pkg/config"
	"poe-helper/pkg/logger"
	"poe-helper/pkg/notify"
	"sync"
)

var (
	cfg      *config.Config
	log      *logger.Logger
	notifier *notify.NotifyService
	initOnce sync.Once
	mu       sync.RWMutex
)

// InitGlobals initializes the global instances of config, logger and notifier.
// This should be called early in the application startup, typically in main().
func InitGlobals(config *config.Config, logger *logger.Logger) {
	initOnce.Do(func() {
		mu.Lock()
		defer mu.Unlock()

		cfg = config
		log = logger
		notifier = notify.NewNotifyService(config.NotifyCommand, logger)
	})
}

// GetConfig returns the global config instance
func GetConfig() *config.Config {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}

// GetLogger returns the global logger instance
func GetLogger() *logger.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return log
}

// GetNotifier returns the global notifier instance
func GetNotifier() *notify.NotifyService {
	mu.RLock()
	defer mu.RUnlock()
	return notifier
}

// GetAll returns all global instances at once.
// This can be useful when multiple services are needed together.
//
//	cfg, log, notifier := global.GetAll()
func GetAll() (*config.Config, *logger.Logger, *notify.NotifyService) {
	mu.RLock()
	defer mu.RUnlock()
	return cfg, log, notifier
}
