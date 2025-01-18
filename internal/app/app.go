package app

import (
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
	"poe-helper/internal/models"
	"poe-helper/internal/poe_log"
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
	p.log.Debug("Initializing main UI",
		"log_path", p.config.LogPath,
		"triggers_count", len(p.config.CompiledTriggers),
	)

	// Create UI components
	p.status = widget.NewLabel("Monitoring POE2...")
	p.entriesContainer = container.NewVBox()
	scrollContainer := container.NewScroll(p.entriesContainer)

	// Create debug panel if in debug mode
	var bottomButtons *fyne.Container

	if p.debugMode {
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

	// Log trigger details
	for name, re := range p.config.CompiledTriggers {
		p.log.Debug("Configured trigger",
			"name", name,
			"pattern", re.String(),
		)
	}

	// Wrap the addLogEntry method to match the new signature
	onTradeEntry := func(entry models.TradeEntry) {
		p.log.Debug("Trade entry received",
			"player", entry.PlayerName,
			"trigger", entry.TriggerType,
			"message", entry.Message,
		)
		p.addLogEntry(TradeEntry{
			Timestamp:   entry.Timestamp,
			TriggerType: entry.TriggerType,
			PlayerName:  entry.PlayerName,
			Message:     entry.Message,
		})
	}

	logWatcher := poe_log.NewLogWatcher(
		p.config.LogPath,
		p.log,
		p.windowFoundTime,
		p.config.CompiledTriggers,
		onTradeEntry,
	)

	p.log.Debug("Creating log watcher",
		"log_path", p.config.LogPath,
		"window_found_time", p.windowFoundTime,
	)

	go logWatcher.Watch()
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
