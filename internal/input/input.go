package input

import (
	"fmt"
	"github.com/go-vgo/robotgo"

	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/logger"
	"hypr-exiled/pkg/notify"

	"hypr-exiled/internal/poe/window"
	"hypr-exiled/internal/wm"
)

type Input struct {
	windowManager *wm.Manager
	detector      *window.Detector
	log           *logger.Logger
	notifier      *notify.NotifyService
}

func NewInput(detector *window.Detector) (*Input, error) {
	log := global.GetLogger()
	notifier := global.GetNotifier()

	return &Input{
		windowManager: detector.GetCurrentWm(),
		detector:      detector,
		log:           log,
		notifier:      notifier,
	}, nil
}

func (i *Input) ExecutePoECommands(commands []string) error {
	if !i.detector.IsActive() {
		return fmt.Errorf("Path of Exile needs to be running")
	}

	window := i.detector.GetCurrentWindow()
	if err := i.windowManager.FocusWindow(window); err != nil {
		return fmt.Errorf("failed to focus window: %w", err)
	}

	for _, cmd := range commands {
		i.log.Debug("Executing PoE command",
			"command", cmd,
			"window_class", window.Class)

		robotgo.KeyTap("enter")
		robotgo.TypeStr(cmd)
		robotgo.KeyTap("enter")
	}
	return nil
}

func (i *Input) ExecuteHideout() error {
	return i.ExecutePoECommands([]string{"/hideout"})
}
