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
	cfg := global.GetConfig()

	if !i.detector.IsActive() {
		return fmt.Errorf("%s needs to be running", cfg.GameNameByAppID(i.detector.ActiveAppID()))
	}

	window := i.detector.GetCurrentWindow()
	if err := i.windowManager.FocusWindow(window); err != nil {
		return fmt.Errorf("failed to focus window: %w", err)
	}

	// Decide profile: PoE1 = slow, PoE2 = fast
	slowTyping := i.isSlowTypingApp()

	if slowTyping {
		// Give PoE1 a moment to accept input after focusing the window.
		time.Sleep(focusDelay)
	}

	for _, cmd := range commands {
		i.log.Debug("Executing PoE command", "command", cmd, "window_class", window.Class)

		if slowTyping {
			// --- SLOW PROFILE (PoE1) ---
			robotgo.KeyTap("enter")     // open chat
			time.Sleep(chatFocusDelay)  // allow input to focus
			robotgo.KeyTap("a", "ctrl") // clear any stale input
			time.Sleep(clearSelectDelay)
			robotgo.KeyTap("backspace")
			time.Sleep(clearDeleteDelay)

			// Type with delay to avoid dropped characters in PoE1.
			robotgo.TypeStrDelay(cmd, typeCharDelayMs)
			time.Sleep(afterTypeDelay)

			robotgo.KeyTap("enter")  // send
			time.Sleep(sendCooldown) // small cooldown between commands
		} else {
			// --- FAST PROFILE (PoE2) ---
			robotgo.KeyTap("enter")
			robotgo.TypeStr(cmd)
			robotgo.KeyTap("enter")
			// No extra sleeps for PoE2
		}
	}
	return nil
}

func (i *Input) ExecuteHideout() error {
	return i.ExecutePoECommands([]string{"/hideout"})
}

func (i *Input) ExecuteKingsmarch() error {
	return i.ExecutePoECommands([]string{"/kingsmarch"})
}

// isSlowTypingApp decides if we should use the slow typing profile.
// Default: PoE1 → slow; PoE2 → fast.
// This avoids magic numbers by resolving via configured game names.
func (i *Input) isSlowTypingApp() bool {
	cfg := global.GetConfig()
	name := cfg.GameNameByAppID(i.detector.ActiveAppID())
	return name == "Path of Exile" // PoE1
}
