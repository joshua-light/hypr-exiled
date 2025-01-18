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

type POEHelper struct {
	config     *Config
	window     fyne.Window
	wm         wm.WindowManager
	log        *logger.Logger
	poeWindow  wm.Window
	hasGame    bool
	ready      bool
	debugPanel *DebugPanel
	debugMode  bool

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

	return &POEHelper{
		config:    config,
		wm:        windowManager,
		log:       log,
		hasGame:   false,
		ready:     false,
		debugMode: debug,
	}, nil
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
			p.poeWindow = window
			p.hasGame = true
			p.mu.Unlock()

			// Only log and initialize UI on initial window discovery
			if !windowFound {
				p.log.Info("Found POE2 window",
					"class", window.Class,
					"title", window.Title,
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
			windowFound = false // Reset the window found flag
			p.mu.Unlock()

			if wasRunning {
				p.log.Info("POE2 window lost")
			}
		}

		// Always keep checking, but with different intervals
		if p.hasGame {
			time.Sleep(5 * time.Second) // Longer interval when window is present
		} else {
			time.Sleep(2 * time.Second) // Shorter interval when looking for window
		}
	}
}

func (p *POEHelper) initializeMainUI() {
	p.log.Debug("Initializing main UI")

	// Create UI components
	p.status = widget.NewLabel("Monitoring POE2...")
	p.logEntries = widget.NewTextGrid()

	// Create debug panel if in debug mode
	if p.debugMode {
		p.debugPanel = NewDebugPanel(p.window)

		// Add debug logging
		debugWriter := NewMemoryLogWriter(p.debugPanel)
		p.log.AddWriter(debugWriter)
	}

	// Create buttons for commands
	buttons := container.NewHBox()
	for cmdName, cmdTemplate := range p.config.Commands {
		cmd := cmdTemplate // Local copy for closure
		btn := widget.NewButton(cmdName, func() {
			if err := p.executeCommand(cmd); err != nil {
				p.log.Error("Failed to execute command", err,
					"command", cmd,
				)
			}
		})
		buttons.Add(btn)
	}

	// Add debug button if in debug mode
	if p.debugMode {
		debugBtn := widget.NewButton("Debug Logs", func() {
			p.debugPanel.Show()
		})
		buttons.Add(debugBtn)
	}

	// Create main layout
	content := container.NewBorder(
		p.status,
		buttons,
		nil,
		nil,
		container.NewScroll(p.logEntries),
	)

	p.window.SetContent(content)
	p.window.Resize(fyne.NewSize(600, 400))

	// Start log watching
	go p.watchLog()
}

func (p *POEHelper) watchLog() {
	p.log.Info("Starting log watcher", "path", p.config.LogPath)

	lastSize := int64(0)
	var lastError error
	var lastErrorTime time.Time

	for {
		p.mu.RLock()
		if p.window == nil {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()

		stat, err := os.Stat(p.config.LogPath)
		if err != nil {
			// Only log error if it's new or if it's been a while since last error
			if lastError == nil || err.Error() != lastError.Error() ||
				time.Since(lastErrorTime) > time.Minute {
				p.log.Error("Failed to stat log file", err)
				lastError = err
				lastErrorTime = time.Now()
			}
			time.Sleep(5 * time.Second)
			continue
		}

		if stat.Size() > lastSize {
			file, err := os.Open(p.config.LogPath)
			if err != nil {
				p.log.Error("Failed to open log file", err)
				time.Sleep(5 * time.Second)
				continue
			}

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
				p.processLogLine(line)
				newLines = append(newLines, line)
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

func (p *POEHelper) processLogLine(line string) {
	p.log.Debug("Processing line", "line", line)

	p.log.Debug("Current triggers", "triggers", p.config.CompiledTriggers)
	for triggerName, re := range p.config.CompiledTriggers {
		p.log.Debug("Trying trigger", "name", triggerName, "pattern", re.String())
		matches := re.FindStringSubmatch(line)
		p.log.Debug("Match result", "matches", matches, "length", len(matches))

		if len(matches) > 1 {
			playerName := matches[1]
			p.log.Info("Triggered event",
				"trigger", triggerName,
				"player", playerName,
				"line", line,
			)

			p.addLogEntry(fmt.Sprintf("[%s] %s: %s",
				time.Now().Format("15:04:05"),
				triggerName,
				playerName,
			))

			// Update UI asynchronously
			go func(cmd string) {
				if cmd, ok := p.config.Commands[triggerName]; ok {
					cmdWithPlayer := strings.ReplaceAll(cmd, "{player}", playerName)
					if err := p.executeCommand(cmdWithPlayer); err != nil {
						p.log.Error("Failed to execute command", err,
							"command", cmdWithPlayer,
							"trigger", triggerName,
							"player", playerName,
						)
					}
				}
			}(triggerName)
		}
	}
}

func (p *POEHelper) addLogEntry(entry string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.log.Debug("Adding log entry", "entry", entry, "textgrid_nil", p.logEntries == nil)

	if p.logEntries == nil {
		return
	}

	// Get current text
	currentText := p.logEntries.Text()

	// Create new text with the new entry at the top
	newText := entry + "\n" + currentText

	// Update the TextGrid
	p.logEntries.SetText(newText)
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
