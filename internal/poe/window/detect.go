package window

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"poe-helper/internal/wm"
	"poe-helper/pkg/logger"
)

// Detector handles POE window detection
type Detector struct {
	log                   *logger.Logger
	poeHelperSessionStart time.Time
	lastResetTimestamp    time.Time
	windowFoundTime       time.Time
	isWindowActive        bool
	mu                    sync.RWMutex
	windowClasses         []string
	windowTitles          []string
	wmManager             *wm.Manager
}

// NewDetector creates a new POE window detector
func NewDetector(log *logger.Logger) *Detector {
	manager, err := wm.NewManager(log)
	if err != nil {
		log.Error("Failed to create window manager", err)
		return nil
	}

	return &Detector{
		log:                   log,
		poeHelperSessionStart: time.Now(),
		windowClasses:         []string{"pathofexile2", "steam_app_PATH_OF_EXILE_2_ID"},
		windowTitles:          []string{"Path of Exile 2", "PoE 2"},
		wmManager:             manager,
	}
}

// Detect checks for the PoE window
func (d *Detector) Detect() error {
	window, err := d.wmManager.FindWindow(d.windowClasses, d.windowTitles)
	if err != nil {
		d.log.Error("Error detecting game window", err)
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	isActive := window != wm.Window{}

	// Window state changed
	if isActive != d.isWindowActive {
		if isActive {
			d.windowFoundTime = time.Now()
			d.log.Info("PoE window found",
				"class", window.Class,
				"title", window.Title)
		} else {
			d.log.Info("PoE window lost")
		}
		d.isWindowActive = isActive
	}

	return nil
}

// CheckLogLineValidity checks if a log line should be processed
func (d *Detector) CheckLogLineValidity(lineTimestamp time.Time, line string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if window is active
	if !d.isWindowActive {
		d.log.Debug("Rejecting line - no active window",
			"line_time", lineTimestamp)
		return false
	}

	// If line is before app started, reject
	if lineTimestamp.Before(d.poeHelperSessionStart) {
		d.log.Debug("Rejecting line - before app start",
			"line_time", lineTimestamp,
			"app_start_time", d.poeHelperSessionStart)
		return false
	}

	// If line is before window was found, reject
	if lineTimestamp.Before(d.windowFoundTime) {
		d.log.Debug("Rejecting line - before window found",
			"line_time", lineTimestamp,
			"window_found_time", d.windowFoundTime)
		return false
	}

	// If "[STARTUP] Loading Start" is seen after app start
	if strings.Contains(line, "[STARTUP] Loading Start") {
		// Update the last reset time to ensure only lines after this are processed
		d.lastResetTimestamp = lineTimestamp

		d.log.Info("Game restart detected",
			"timestamp", lineTimestamp)
		return false
	}

	// If last reset timestamp exists, only process lines after it
	if !d.lastResetTimestamp.IsZero() && lineTimestamp.Before(d.lastResetTimestamp) {
		d.log.Debug("Rejecting line - before game restart",
			"line_time", lineTimestamp,
			"restart_time", d.lastResetTimestamp)
		return false
	}

	// Only process lines with "@From" or "@To"
	if !strings.Contains(line, "@From") && !strings.Contains(line, "@To") {
		d.log.Debug("Rejecting line - not a trade message")
		return false
	}

	return true
}

// Start begins monitoring window state
func (d *Detector) Start() error {
	if d.wmManager == nil {
		return fmt.Errorf("window manager not initialized")
	}

	go func() {
		for {
			if err := d.Detect(); err != nil {
				d.log.Error("Window detection error", err)
			}
			time.Sleep(2 * time.Second)
		}
	}()
	return nil
}
