package app

import (
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"poe-helper/pkg/logger"
)

// DebugPanel represents the debug window and its components
type DebugPanel struct {
	window    fyne.Window
	textArea  *widget.TextGrid
	logger    *logger.Logger
	mu        sync.Mutex
	content   []string
	isVisible bool
}

func NewDebugPanel(parent fyne.Window, log *logger.Logger) *DebugPanel {
	dp := &DebugPanel{
		logger:    log,
		content:   make([]string, 0),
		isVisible: false,
	}

	// Create a simple window
	dp.window = fyne.CurrentApp().NewWindow("Debug")

	// Create a basic text display
	dp.textArea = widget.NewTextGrid()

	// Create buttons
	testBtn := widget.NewButton("Test Log", func() {
		dp.logger.Debug("Test log entry from debug panel")
	})

	clearBtn := widget.NewButton("Clear", func() {
		dp.Clear()
	})

	// Create button container
	buttons := container.NewHBox(testBtn, clearBtn)

	// Create a basic layout with scroll
	scrollContainer := container.NewScroll(dp.textArea)
	content := container.NewBorder(
		buttons, // top
		nil,     // bottom
		nil,     // left
		nil,     // right
		scrollContainer,
	)

	dp.window.SetContent(content)
	dp.window.Resize(fyne.NewSize(800, 600))

	// Set up close handling
	dp.window.SetCloseIntercept(func() {
		dp.Hide()
	})

	// Initial text
	dp.AddText("Debug Panel Initialized")

	return dp
}

func (dp *DebugPanel) AddText(text string) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if dp.textArea == nil {
		return
	}

	// Add new line to content
	dp.content = append(dp.content, text)

	// Keep only last 1000 lines
	if len(dp.content) > 1000 {
		dp.content = dp.content[len(dp.content)-1000:]
	}

	// Update the text grid
	displayText := strings.Join(dp.content, "\n")
	dp.textArea.SetText(displayText)
	dp.window.Canvas().Refresh(dp.textArea)
}

func (dp *DebugPanel) Clear() {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.content = make([]string, 0)
	if dp.textArea != nil {
		dp.textArea.SetText("")
		dp.window.Canvas().Refresh(dp.textArea)
	}
}

func (dp *DebugPanel) Show() {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if dp.window != nil {
		dp.isVisible = true
		dp.window.Show()
	}
}

func (dp *DebugPanel) Hide() {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if dp.window != nil {
		dp.isVisible = false
		dp.window.Hide()
	}
}

func (dp *DebugPanel) IsVisible() bool {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	return dp.isVisible
}

// DebugWriter implements io.Writer for logging
type DebugWriter struct {
	panel *DebugPanel
}

func NewDebugWriter(panel *DebugPanel) *DebugWriter {
	writer := &DebugWriter{panel: panel}
	return writer
}

func (w *DebugWriter) Write(p []byte) (n int, err error) {
	text := strings.TrimSpace(string(p))
	if text != "" {
		w.panel.AddText(text)
	}
	return len(p), nil
}
