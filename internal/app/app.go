package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"poe-helper/internal/input"
	"poe-helper/internal/wm"
	"poe-helper/pkg/logger"
)

type TradeEntry struct {
	Timestamp   time.Time
	TriggerType string
	PlayerName  string
	Message     string
}

type POEHelper struct {
	config           *Config
	window           fyne.Window
	wm               wm.WindowManager
	log              *logger.Logger
	poeWindow        wm.Window
	hasGame          bool
	ready            bool
	debugPanel       *DebugPanel
	debugMode        bool
	windowFoundTime  time.Time
	sessionStartTime time.Time
	foundSession     bool

	// Trade entries
	tradeEntries     []TradeEntry
	entriesContainer *fyne.Container

	// UI elements
	status     *widget.Label
	logEntries *widget.TextGrid

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

func NewPOEHelper(config *Config, log *logger.Logger, debug bool) (*POEHelper, error) {
	log.Debug("Initializing POE Helper", "debug_mode", debug)

	if config == nil {
		config = DefaultConfig()
		log.Info("Using default configuration")
	}

	// Set default log path if not specified
	if config.LogPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		// Common POE2 log paths
		possiblePaths := []string{
			filepath.Join(home, ".local", "share", "Steam", "steamapps", "common", "Path of Exile 2", "logs", "Client.txt"),
			filepath.Join(home, "Games", "Path of Exile 2", "logs", "Client.txt"),
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				config.LogPath = path
				log.Info("Found POE2 log file", "path", path)
				break
			}
		}

		if config.LogPath == "" {
			log.Warn("Could not find POE2 log file in common locations")
		}
	}

	// Try to detect window manager
	var windowManager wm.WindowManager
	var err error

	log.Debug("Detecting window manager")

	windowManager, err = wm.NewHyprland()
	if err != nil {
		// If Hyprland fails, try X11
		windowManager, err = wm.NewX11()
		if err != nil {
			return nil, fmt.Errorf("no supported window manager found")
		}
		log.Info("Using X11 window manager")
	} else {
		log.Info("Using Hyprland window manager")
	}

	helper := &POEHelper{
		config:    config,
		wm:        windowManager,
		log:       log,
		hasGame:   false,
		ready:     false,
		debugMode: debug,
	}

	return helper, nil
}

func (p *POEHelper) Run() error {
	p.log.Info("Starting POE Helper application")

	a := app.New()
	p.window = a.NewWindow("PoE Helper")

	// Show waiting screen
	p.showWaitingScreen()

	// Start game detection
	go p.detectGame()

	p.window.ShowAndRun()
	return nil
}

func (p *POEHelper) showWaitingScreen() {
	p.log.Debug("Showing waiting screen")
	content := container.NewCenter(
		container.NewVBox(
			widget.NewLabel("Waiting for Path of Exile 2..."),
			widget.NewProgressBarInfinite(),
		),
	)
	p.window.SetContent(content)
	p.window.Resize(fyne.NewSize(300, 150))
}

func (p *POEHelper) detectGame() {
	p.log.Debug("Starting game detection loop")
	windowFound := false

	for {
		p.mu.RLock()
		if p.window == nil {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()

		window, err := p.wm.FindWindow(
			[]string{"pathofexile2", "steam_app_PATH_OF_EXILE_2_ID"},
			[]string{"Path of Exile 2", "PoE 2"},
		)

		if err != nil {
			p.log.Error("Error detecting game window", err)
		} else if window != (wm.Window{}) {
			p.mu.Lock()
			if !p.hasGame {
				// Record timestamp when window is first found
				p.windowFoundTime = time.Now()
				p.foundSession = false // Reset session detection
			}
			p.poeWindow = window
			p.hasGame = true
			p.mu.Unlock()

			// Only log and initialize UI on initial window discovery
			if !windowFound {
				p.log.Info("Found POE2 window",
					"class", window.Class,
					"title", window.Title,
					"time", p.windowFoundTime,
				)

				// Initialize main UI if not ready
				if !p.ready {
					p.initializeMainUI()
					p.ready = true
				}
				windowFound = true
			}
		} else {
			p.mu.Lock()
			wasRunning := p.hasGame
			p.hasGame = false
			windowFound = false
			p.windowFoundTime = time.Time{}
			p.foundSession = false
			p.mu.Unlock()

			if wasRunning {
				p.log.Info("POE2 window lost")
			}
		}

		if p.hasGame {
			time.Sleep(5 * time.Second)
		} else {
			time.Sleep(2 * time.Second)
		}
	}
}

func (p *POEHelper) initializeMainUI() {
	p.log.Debug("Initializing main UI")

	// Create UI components
	p.status = widget.NewLabel("Monitoring POE2...")
	p.entriesContainer = container.NewVBox()
	scrollContainer := container.NewScroll(p.entriesContainer)

	// Create debug panel if in debug mode
	var bottomButtons *fyne.Container

	if p.debugMode {
		// Now we have a valid window to pass
		p.debugPanel = NewDebugPanel(p.window, p.log)
		debugBtn := widget.NewButton("Debug", func() {
			if !p.debugPanel.IsVisible() {
				p.debugPanel.Show()
			}
		})
		bottomButtons = container.NewHBox(debugBtn)
	}

	// Create main layout with padding
	content := container.NewPadded(
		container.NewBorder(
			p.status,
			bottomButtons,
			nil,
			nil,
			scrollContainer,
		),
	)

	p.window.SetContent(content)
	p.window.Resize(fyne.NewSize(600, 400))

	// Add initial debug log
	p.log.Debug("Main UI initialized",
		"container_nil", p.entriesContainer == nil,
	)
	// Start log watching
	go p.watchLog()
}

func (p *POEHelper) watchLog() {
	p.log.Info("Starting log watcher", "path", p.config.LogPath)

	// Get initial file size
	stat, err := os.Stat(p.config.LogPath)
	if err != nil {
		p.log.Error("Failed to stat log file", err)
		return
	}

	// Start from the end of the file
	lastSize := stat.Size()
	var lastError error
	var lastErrorTime time.Time

	p.log.Debug("Starting log watch from offset",
		"size", lastSize,
		"path", p.config.LogPath,
	)

	for {
		p.mu.RLock()
		if p.window == nil {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()

		stat, err := os.Stat(p.config.LogPath)
		if err != nil {
			if lastError == nil || err.Error() != lastError.Error() ||
				time.Since(lastErrorTime) > time.Minute {
				p.log.Error("Failed to stat log file", err)
				lastError = err
				lastErrorTime = time.Now()
			}
			time.Sleep(5 * time.Second)
			continue
		}

		// If file is truncated or rotated, reset to beginning
		if stat.Size() < lastSize {
			p.log.Info("Log file was truncated, resetting position",
				"old_size", lastSize,
				"new_size", stat.Size(),
			)
			lastSize = 0
		}

		if stat.Size() > lastSize {
			file, err := os.Open(p.config.LogPath)
			if err != nil {
				p.log.Error("Failed to open log file", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// Seek to last read position
			if lastSize > 0 {
				_, err = file.Seek(lastSize, 0)
				if err != nil {
					p.log.Error("Failed to seek in log file", err)
					file.Close()
					continue
				}
			}

			newLines := make([]string, 0)
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				// Extract timestamp from the log line
				timestamp, err := time.Parse("2006/01/02 15:04:05", line[:19])
				if err != nil {
					p.log.Debug("Failed to parse timestamp from line",
						"line", line,
						"error", err,
					)
					continue
				}

				p.mu.RLock()
				windowTime := p.windowFoundTime
				p.mu.RUnlock()

				// Only process lines that occurred after we found the window
				if !windowTime.IsZero() && timestamp.After(windowTime) {
					p.processLogLine(line)
					newLines = append(newLines, line)
				}
			}

			file.Close()
			lastSize = stat.Size()

			if len(newLines) > 0 {
				p.log.Debug("Processed new log lines",
					"count", len(newLines),
					"last_line", newLines[len(newLines)-1],
				)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (p *POEHelper) shouldProcessLine(line string) bool {
	// Extract timestamp from the log line
	timestamp, err := time.Parse("2006/01/02 15:04:05", line[:19])
	if err != nil {
		p.log.Debug("Failed to parse timestamp from line", "line", line, "error", err)
		return false
	}

	p.mu.RLock()
	windowTime := p.windowFoundTime
	foundSession := p.foundSession
	p.mu.RUnlock()

	// If we haven't found the session start yet, look for it
	if !foundSession {
		if strings.Contains(line, "[STARTUP] Loading Start") && timestamp.After(windowTime) {
			p.mu.Lock()
			p.sessionStartTime = timestamp
			p.foundSession = true
			p.mu.Unlock()

			p.log.Info("Found new POE session start",
				"time", timestamp,
				"window_found_time", windowTime,
			)
			return true
		}
		return false
	}

	// Only process lines after session start
	return timestamp.After(p.sessionStartTime) || timestamp.Equal(p.sessionStartTime)
}

func (p *POEHelper) processLogLine(line string) {
	// Initial timestamp validation
	timestamp, err := time.Parse("2006/01/02 15:04:05", line[:19])
	if err != nil {
		p.log.Debug("Failed to parse timestamp from line", "line", line, "error", err)
		return
	}

	// Window time validation with proper locking
	p.mu.RLock()
	windowTime := p.windowFoundTime
	p.mu.RUnlock()

	if !windowTime.IsZero() && timestamp.Before(windowTime) {
		p.log.Debug("Skipping old log line",
			"line_time", timestamp,
			"window_time", windowTime,
		)
		return
	}

	p.log.Debug("Processing line", "line", line)

	// Process trade messages
	for triggerName, re := range p.config.CompiledTriggers {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			playerName := matches[1]
			p.log.Info("Triggered event",
				"trigger", triggerName,
				"player", playerName,
				"line", line,
			)

			// Create the trade entry
			entry := TradeEntry{
				Timestamp:   timestamp, // Use the parsed timestamp from the log
				TriggerType: triggerName,
				PlayerName:  playerName,
				Message:     line,
			}

			// Since we're running in a goroutine from watchLog(),
			// we need to safely update the UI
			if p.window != nil {
				// Create UI elements for this trade
				messageLabel := widget.NewLabel(fmt.Sprintf("[%s] %s: %s",
					entry.Timestamp.Format("15:04:05"),
					entry.TriggerType,
					entry.PlayerName,
				))

				// Create action buttons
				tradeBtn := widget.NewButton("trade", func() {
					if cmd, ok := p.config.Commands["trade"]; ok {
						cmdWithPlayer := strings.ReplaceAll(cmd, "{player}", entry.PlayerName)
						p.executeCommand(cmdWithPlayer)
					}
				})

				inviteBtn := widget.NewButton("invite", func() {
					if cmd, ok := p.config.Commands["invite"]; ok {
						cmdWithPlayer := strings.ReplaceAll(cmd, "{player}", entry.PlayerName)
						p.executeCommand(cmdWithPlayer)
					}
				})

				thankBtn := widget.NewButton("thank", func() {
					if cmd, ok := p.config.Commands["thank"]; ok {
						cmdWithPlayer := strings.ReplaceAll(cmd, "{player}", entry.PlayerName)
						p.executeCommand(cmdWithPlayer)
					}
				})

				buttonBox := container.NewHBox(tradeBtn, inviteBtn, thankBtn)
				entryBox := container.NewVBox(messageLabel, buttonBox)

				p.mu.Lock()

				if p.entriesContainer != nil {
					p.entriesContainer.Add(entryBox)
					p.window.Canvas().Refresh(p.entriesContainer)
				}
				p.mu.Unlock()
			}
		}
	}
}

func (p *POEHelper) addLogEntry(entry TradeEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.log.Debug("Adding trade entry",
		"player", entry.PlayerName,
		"container_nil", p.entriesContainer == nil,
	)

	if p.entriesContainer == nil {
		p.log.Error("Entries container is nil", fmt.Errorf("container not initialized"))
		return
	}

	p.tradeEntries = append(p.tradeEntries, entry)

	// Create entry container with message and buttons
	messageLabel := widget.NewLabel(fmt.Sprintf("[%s] %s: %s",
		entry.Timestamp.Format("15:04:05"),
		entry.TriggerType,
		entry.PlayerName,
	))

	// Create buttons for this specific trade
	tradeBtn := widget.NewButton("trade", func() {
		p.log.Debug("Trade button clicked", "player", entry.PlayerName)
		if cmd, ok := p.config.Commands["trade"]; ok {
			cmdWithPlayer := strings.ReplaceAll(cmd, "{player}", entry.PlayerName)
			p.executeCommand(cmdWithPlayer)
		}
	})

	inviteBtn := widget.NewButton("invite", func() {
		p.log.Debug("Invite button clicked", "player", entry.PlayerName)
		if cmd, ok := p.config.Commands["invite"]; ok {
			cmdWithPlayer := strings.ReplaceAll(cmd, "{player}", entry.PlayerName)
			p.executeCommand(cmdWithPlayer)
		}
	})

	thankBtn := widget.NewButton("thank", func() {
		p.log.Debug("Thank button clicked", "player", entry.PlayerName)
		if cmd, ok := p.config.Commands["thank"]; ok {
			cmdWithPlayer := strings.ReplaceAll(cmd, "{player}", entry.PlayerName)
			p.executeCommand(cmdWithPlayer)
		}
	})

	buttonBox := container.NewHBox(tradeBtn, inviteBtn, thankBtn)
	entryBox := container.NewVBox(
		messageLabel,
		buttonBox,
	)

	// Make sure the entry is visible
	entryBox.Resize(fyne.NewSize(580, 80))

	p.log.Debug("Adding new entry to container")

	// Use Fyne's Refresh() to ensure the UI updates
	p.entriesContainer.Add(entryBox)
	p.entriesContainer.Refresh()
}

func (p *POEHelper) executeCommand(cmd string) error {
	p.mu.RLock()
	if !p.hasGame {
		p.mu.RUnlock()
		return fmt.Errorf("POE2 window not found")
	}
	window := p.poeWindow
	p.mu.RUnlock()

	p.log.Debug("Executing command",
		"command", cmd,
		"window", window,
	)

	if err := p.wm.FocusWindow(window); err != nil {
		return fmt.Errorf("failed to focus POE2 window: %w", err)
	}

	if err := input.ExecuteInput(cmd); err != nil {
		return fmt.Errorf("failed to execute input: %w", err)
	}

	return nil
}

// Cleanup performs cleanup operations before shutting down
func (p *POEHelper) Cleanup() {
	p.log.Info("Cleaning up POE Helper")

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.debugPanel != nil {
		p.debugPanel.Hide()
	}

	// Set window to nil to signal goroutines to stop
	p.window = nil
}
