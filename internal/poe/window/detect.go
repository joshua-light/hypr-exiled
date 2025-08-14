package window

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"hypr-exiled/internal/wm"
	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/notify"
)

// Detector handles POE window detection
type Detector struct {
	hyprExiledSessionStart time.Time
	lastResetTimestamp     time.Time
	windowFoundTime        time.Time
	isWindowActive         bool
	currentWindow          wm.Window
	mu                     sync.RWMutex
	windowClasses          []string
	wmManager              *wm.Manager
	stopChan               chan struct{}
	stopped                bool
	activeAppID            int
	changeChan             chan int
}

// NewDetector creates a new POE window detector
func NewDetector() *Detector {
	log := global.GetLogger()

	manager, err := wm.NewManager()
	if err != nil {
		log.Error("Failed to create window manager", err)
		return nil
	}

	cfg := global.GetConfig()

	return &Detector{
		hyprExiledSessionStart: time.Now(),
		windowClasses:          cfg.WindowClasses(),
		wmManager:              manager,
		stopChan:               make(chan struct{}),

		activeAppID: cfg.GetDefaultAppID(),
		changeChan:  make(chan int, 1),
	}
}

func (d *Detector) Changes() <-chan int { return d.changeChan }
func (d *Detector) ActiveAppID() int    { d.mu.RLock(); defer d.mu.RUnlock(); return d.activeAppID }

func parseSteamAppID(class string) int {
	const pref = "steam_app"
	if !strings.HasPrefix(class, pref) {
		return 0
	}

	id, _ := strconv.Atoi(strings.TrimPrefix(class, pref))
	return id
}

// Detect checks for the PoE window
func (d *Detector) Detect() error {
	log := global.GetLogger()
	notifier := global.GetNotifier()
	cfg := global.GetConfig()

	window, err := d.wmManager.FindWindow(d.windowClasses)
	if err != nil {
		log.Error("Error detecting game window", err)
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.currentWindow = window // Store window
	isActive := window != wm.Window{}

	if isActive != d.isWindowActive {
		if isActive {
			d.windowFoundTime = time.Now()
			log.Info("PoE window found", "class", window.Class)
			notifier.Show("PoE window found, monitoring trades...", notify.Info)
		} else {
			log.Info("PoE window lost")
			notifier.Show("PoE window lost", notify.Info)
		}
		d.isWindowActive = isActive
	}

	if isActive && window.Class != "" {
		var id int
		if v, ok := cfg.AppIDByWindowClass(window.Class); ok {
			id = v
		} else {
			id = parseSteamAppID(window.Class) // fallback
		}

		if id != 0 && id != d.activeAppID {
			old := d.activeAppID
			d.activeAppID = id
			d.windowFoundTime = time.Now() // optional: from now on allow new lines
			log.Info("Detected different PoE variant",
				"old_app_id", old,
				"new_app_id", id,
				"class", window.Class)
			select {
			case d.changeChan <- id:
			default:
				//not blocking
			}
		}
	}

	return nil
}

// CheckLogLineValidity checks if a log line should be processed
func (d *Detector) CheckLogLineValidity(lineTimestamp time.Time, line string) bool {
	log := global.GetLogger()
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if window is active
	if !d.isWindowActive {
		log.Debug("Rejecting line - no active window",
			"line_time", lineTimestamp)
		return false
	}

	// If line is before app started, reject
	if lineTimestamp.Before(d.hyprExiledSessionStart) {
		log.Debug("Rejecting line - before app start",
			"line_time", lineTimestamp,
			"app_start_time", d.hyprExiledSessionStart)
		return false
	}

	// If line is before window was found, reject
	if lineTimestamp.Before(d.windowFoundTime) {
		log.Debug("Rejecting line - before window found",
			"line_time", lineTimestamp,
			"window_found_time", d.windowFoundTime)
		return false
	}

	// If "[STARTUP] Loading Start" is seen after app start
	if strings.Contains(line, "[STARTUP] Loading Start") {
		// Update the last reset time to ensure only lines after this are processed
		d.lastResetTimestamp = lineTimestamp

		log.Info("Game restart detected",
			"timestamp", lineTimestamp)
		return false
	}

	// If last reset timestamp exists, only process lines after it
	if !d.lastResetTimestamp.IsZero() && lineTimestamp.Before(d.lastResetTimestamp) {
		log.Debug("Rejecting line - before game restart",
			"line_time", lineTimestamp,
			"restart_time", d.lastResetTimestamp)
		return false
	}

	// Only process lines with "@From" or "@To"
	if !strings.Contains(line, "@From") && !strings.Contains(line, "@To") {
		log.Debug("Rejecting line - not a trade message")
		return false
	}

	return true
}

// Start begins monitoring window state
func (d *Detector) Start() error {
	log := global.GetLogger()

	if d.wmManager == nil {
		return fmt.Errorf("window manager not initialized")
	}

	d.mu.Lock()
	if d.stopped {
		d.stopChan = make(chan struct{})
		d.stopped = false
	}
	d.mu.Unlock()

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-d.stopChan:
				log.Info("Window detector stopped")
				return
			case <-ticker.C:
				if err := d.Detect(); err != nil {
					log.Error("Window detection error", err)
				}
			}
		}
	}()

	log.Info("Window detector started")
	return nil
}

func (d *Detector) GetCurrentWindow() wm.Window {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.currentWindow
}

func (d *Detector) GetCurrentWm() *wm.Manager {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.wmManager
}

func (d *Detector) IsActive() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.isWindowActive
}

// Stop stops the window detection loop
func (d *Detector) Stop() error {
	log := global.GetLogger()

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		log.Debug("Window detector already stopped")
		return nil
	}

	log.Info("Stopping window detector")
	close(d.stopChan)
	d.stopped = true

	return nil
}
