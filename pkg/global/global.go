package global

import (
	"embed"
	"sync"

	"hypr-exiled/pkg/config"
	"hypr-exiled/pkg/logger"
	"hypr-exiled/pkg/notify"
	"hypr-exiled/pkg/sound"
)

var (
	cfg           *config.Config
	log           *logger.Logger
	notifier      *notify.NotifyService
	soundNotifier *sound.SoundNotifier
	initOnce      sync.Once
	mu            sync.RWMutex
)

func InitGlobals(config *config.Config, logger *logger.Logger, assets embed.FS) {
	initOnce.Do(func() {
		mu.Lock()
		defer mu.Unlock()
		cfg = config
		log = logger
		notifier = notify.NewNotifyService(config.GetNotifyCommand(), logger)

		sn, err := sound.NewSoundNotifier(assets)
		if err != nil {
			logger.Error("Failed to initialize sound notifier", err)
		} else {
			soundNotifier = sn
		}
	})
}

func GetSoundNotifier() *sound.SoundNotifier {
	mu.RLock()
	defer mu.RUnlock()
	return soundNotifier
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
func GetAll() (*config.Config, *logger.Logger, *notify.NotifyService) {
	mu.RLock()
	defer mu.RUnlock()
	return cfg, log, notifier
}
