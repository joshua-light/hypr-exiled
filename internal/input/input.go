package input

import (
	"fmt"
	"time"

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

// Typing/timing parameters (tune as needed; consider moving to config later).
const (
	focusDelay       = 150 * time.Millisecond // after focusing the game window
	chatFocusDelay   = 100 * time.Millisecond // after opening chat
	clearSelectDelay = 30 * time.Millisecond  // after Ctrl+A
	clearDeleteDelay = 30 * time.Millisecond  // after Backspace
	afterTypeDelay   = 40 * time.Millisecond  // after typing the command
	sendCooldown     = 120 * time.Millisecond // between consecutive commands

	typeCharDelayMs = 10 // per-character typing delay for robotgo.TypeStrDelay
)

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

	// Give PoE a moment to accept input after focusing the window.
	time.Sleep(focusDelay)

	for _, cmd := range commands {
		i.log.Debug("Executing PoE command", "command", cmd, "window_class", window.Class)

		// Open chat.
		robotgo.KeyTap("enter")
		time.Sleep(chatFocusDelay) // allow the chat input to focus

		// Safety: clear any existing text to avoid sending stale content.
		robotgo.KeyTap("a", "ctrl") // select all
		time.Sleep(clearSelectDelay)
		robotgo.KeyTap("backspace") // delete selection
		time.Sleep(clearDeleteDelay)

		// Type slowly: PoE (1) can drop initial characters if typing is too fast.
		// Depending on robotgo version this may be TypeStrDelay or TypeStrDelayed.
		robotgo.TypeStrDelay(cmd, typeCharDelayMs)
		time.Sleep(afterTypeDelay)

		// Send the command.
		robotgo.KeyTap("enter")
		time.Sleep(sendCooldown) // small cooldown between commands
	}
	return nil
}

func (i *Input) ExecuteHideout() error {
	return i.ExecutePoECommands([]string{"/hideout"})
}
