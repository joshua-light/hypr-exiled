package app

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// DebugPanel represents the debug UI window
type DebugPanel struct {
	mu          sync.RWMutex
	logText     *widget.TextGrid // Changed from TextGrid to TextArea for better performance
	window      fyne.Window
	maxLines    int
	lines       []string
	autoscroll  bool
	filter      string
	filterEntry *widget.Entry
}

// NewDebugPanel creates a new debug panel
func NewDebugPanel(parent fyne.Window) *DebugPanel {
	dp := &DebugPanel{
		maxLines:   1000,
		lines:      make([]string, 0, 1000),
		autoscroll: true,
	}

	dp.window = fyne.CurrentApp().NewWindow("POE Helper Debug")

	// Create log text area
	dp.logText = widget.NewTextGrid()

	// Create filter input
	dp.filterEntry = widget.NewEntry()
	dp.filterEntry.SetPlaceHolder("Filter logs...")
	dp.filterEntry.OnChanged = func(value string) {
		dp.setFilter(value)
	}

	// Create autoscroll toggle
	autoscrollCheck := widget.NewCheck("Autoscroll", func(value bool) {
		dp.setAutoscroll(value)
	})
	autoscrollCheck.SetChecked(true)

	// Create clear button
	clearBtn := widget.NewButton("Clear", dp.clear)

	// Create controls container
	controls := container.NewHBox(
		dp.filterEntry,
		autoscrollCheck,
		clearBtn,
	)

	// Layout
	content := container.NewBorder(
		controls,
		nil,
		nil,
		nil,
		container.NewScroll(dp.logText),
	)

	dp.window.SetContent(content)
	dp.window.Resize(fyne.NewSize(800, 400))

	return dp
}

// Show displays the debug window
func (dp *DebugPanel) Show() {
	dp.window.Show()
}

// Hide hides the debug window
func (dp *DebugPanel) Hide() {
	dp.window.Hide()
}

// AddLogLine adds a new log entry to the debug panel
func (dp *DebugPanel) AddLogLine(line string) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	// Add timestamp if not present
	if !strings.Contains(line, "202") { // Simple check for timestamp
		line = fmt.Sprintf("%s %s", time.Now().Format("2006-01-02T15:04:05.000Z"), line)
	}

	// Add to lines buffer
	dp.lines = append(dp.lines, line)

	// Trim if exceeds max lines
	if len(dp.lines) > dp.maxLines {
		dp.lines = dp.lines[len(dp.lines)-dp.maxLines:]
	}

	// Update display
	dp.updateDisplay()
}

// clear clears all log entries
func (dp *DebugPanel) clear() {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.lines = make([]string, 0, dp.maxLines)
	dp.updateDisplay()
}

// setFilter updates the log filter
func (dp *DebugPanel) setFilter(filter string) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.filter = strings.ToLower(filter)
	dp.updateDisplay()
}

// setAutoscroll toggles autoscroll
func (dp *DebugPanel) setAutoscroll(value bool) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.autoscroll = value
}

// updateDisplay updates the display with filtered logs
func (dp *DebugPanel) updateDisplay() {
	// Filter lines
	var displayLines []string
	if dp.filter != "" {
		displayLines = make([]string, 0, len(dp.lines))
		for _, line := range dp.lines {
			if strings.Contains(strings.ToLower(line), dp.filter) {
				displayLines = append(displayLines, line)
			}
		}
	} else {
		displayLines = dp.lines
	}

	// Join lines and update TextGrid
	text := strings.Join(displayLines, "\n")
	dp.logText.SetText(text)
}

// MemoryLogWriter implements io.Writer for capturing logs
type MemoryLogWriter struct {
	debugPanel *DebugPanel
}

// NewMemoryLogWriter creates a new memory log writer
func NewMemoryLogWriter(dp *DebugPanel) *MemoryLogWriter {
	return &MemoryLogWriter{debugPanel: dp}
}

// Write implements io.Writer
func (w *MemoryLogWriter) Write(p []byte) (n int, err error) {
	w.debugPanel.AddLogLine(string(p))
	return len(p), nil
}
