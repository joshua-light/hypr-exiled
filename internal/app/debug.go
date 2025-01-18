package app

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"poe-helper/pkg/logger"
)

type DebugPanel struct {
	mu              sync.RWMutex
	logText         *widget.RichText
	window          fyne.Window
	maxLines        int
	lines           []string
	filter          string
	filterEntry     *widget.Entry
	isVisible       bool
	logger          *logger.Logger
	scrollContainer *container.Scroll
	lineCount       int
}

func NewDebugPanel(parent fyne.Window, log *logger.Logger) *DebugPanel {
	if parent == nil {
		log.Error("Cannot create debug panel: parent window is nil", fmt.Errorf("parent window is nil"))
		return nil
	}

	dp := &DebugPanel{
		maxLines:  1000,
		lines:     make([]string, 0, 1000),
		isVisible: false,
		logger:    log,
	}

	dp.logger.Debug("Creating new DebugPanel instance")

	dp.window = fyne.CurrentApp().NewWindow("POE Helper Debug")
	dp.logText = widget.NewRichText()
	dp.logText.Wrapping = fyne.TextWrapBreak

	dp.filterEntry = widget.NewEntry()
	dp.filterEntry.SetPlaceHolder("Filter logs...")
	dp.filterEntry.OnChanged = func(value string) {
		dp.setFilter(value)
	}

	clearBtn := widget.NewButton("Clear", func() { dp.clear() })
	testBtn := widget.NewButton("Test Log", func() {
		dp.logger.Debug("Test log entry from debug panel")
	})

	controls := container.NewHBox(
		dp.filterEntry,
		clearBtn,
		testBtn,
	)

	dp.scrollContainer = container.NewScroll(dp.logText)
	content := container.NewBorder(
		controls,
		nil,
		nil,
		nil,
		dp.scrollContainer,
	)

	dp.window.SetContent(content)
	dp.window.Resize(fyne.NewSize(800, 400))

	dp.window.SetCloseIntercept(func() {
		dp.logger.Debug("Window close intercepted")
		dp.Hide()
	})

	// Add initial lines
	dp.AddLogLine("Debug Panel Initialized")
	dp.AddLogLine("MemoryLogWriter initialized")

	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			if dp.IsVisible() {
				dp.ForceRefresh()
			}
		}
	}()

	return dp
}

func (dp *DebugPanel) Show() {
	dp.mu.Lock()
	dp.isVisible = true
	dp.mu.Unlock()

	dp.window.Show()
	dp.updateDisplay()
}

func (dp *DebugPanel) Hide() {
	dp.mu.Lock()
	dp.isVisible = false
	dp.mu.Unlock()

	dp.window.Hide()
}

func (dp *DebugPanel) IsVisible() bool {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	return dp.isVisible
}

func (dp *DebugPanel) clear() {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.lines = make([]string, 0, dp.maxLines)
	dp.logText.Segments = []widget.RichTextSegment{}
	dp.logText.Refresh()
}

func (dp *DebugPanel) setFilter(filter string) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.filter = strings.ToLower(filter)
	dp.updateDisplay()
}

func (dp *DebugPanel) AddLogLine(line string) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	// Clean up the line
	line = strings.TrimSpace(line)

	// Don't add timestamp if line already has one
	if !strings.HasPrefix(line, "202") { // Year prefix check
		line = fmt.Sprintf("%s %s", time.Now().Format("2006/01/02 15:04:05"), line)
	}

	dp.lines = append(dp.lines, line)

	// Trim if exceeds max lines
	if len(dp.lines) > dp.maxLines {
		dp.lines = dp.lines[len(dp.lines)-dp.maxLines:]
	}

	// Always update display when a new line is added if visible
	if dp.isVisible {
		dp.updateDisplayLocked()
	}
}

func (dp *DebugPanel) updateDisplayLocked() {
	if dp.logText == nil {
		return
	}

	var displayLines []string
	if dp.filter != "" {
		displayLines = make([]string, 0, len(dp.lines))
		for _, line := range dp.lines {
			if strings.Contains(strings.ToLower(line), strings.ToLower(dp.filter)) {
				displayLines = append(displayLines, line)
			}
		}
	} else {
		displayLines = dp.lines
	}

	// Create new segments for each line
	segments := make([]widget.RichTextSegment, len(displayLines))
	for i, line := range displayLines {
		segments[i] = &widget.TextSegment{
			Text: line + "\n",
		}
	}

	dp.logText.Segments = segments

	// Force refresh of the widget and its container
	dp.logText.Refresh()
	if dp.scrollContainer != nil {
		dp.scrollContainer.ScrollToBottom()
		dp.scrollContainer.Refresh()
	}
}

func (dp *DebugPanel) updateDisplay() {
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

	// Create new segments for each line
	segments := make([]widget.RichTextSegment, len(displayLines))
	for i, line := range displayLines {
		segments[i] = &widget.TextSegment{
			Text: line + "\n",
		}
	}

	dp.logText.Segments = segments
	dp.logText.Refresh()
	dp.scrollContainer.ScrollToBottom()
}

type MemoryLogWriter struct {
	debugPanel *DebugPanel
	logger     *logger.Logger
}

func NewMemoryLogWriter(dp *DebugPanel, log *logger.Logger) *MemoryLogWriter {
	writer := &MemoryLogWriter{
		debugPanel: dp,
		logger:     log,
	}
	writer.Write([]byte("MemoryLogWriter initialized"))
	return writer
}

func (dp *DebugPanel) ForceRefresh() {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if dp.isVisible {
		dp.updateDisplayLocked()
	}
}

func (w *MemoryLogWriter) Write(p []byte) (n int, err error) {
	text := strings.TrimSpace(string(p))
	if text == "" {
		return len(p), nil
	}
	// The log lines come in with ISO8601 timestamps, we need to parse and reformat them
	// Example: 2025-01-18T17:49:44+01:00 DBG Test log entry...
	if len(text) > 20 { // Basic length check for timestamp
		line := text
		// Don't modify the timestamp if it's already in the desired format
		if !strings.HasPrefix(line, "2025/") {
			timestamp := text[:25] // Extract ISO timestamp
			message := text[26:]   // Rest of the message

			// Parse the ISO timestamp
			if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
				// Reformat to match your desired format
				line = fmt.Sprintf("%s %s", t.Format("2006/01/02 15:04:05"), message)
			}
		}

		w.debugPanel.mu.Lock()
		shouldUpdate := w.debugPanel.isVisible
		w.debugPanel.lines = append(w.debugPanel.lines, line)

		// Trim if exceeds max lines
		if len(w.debugPanel.lines) > w.debugPanel.maxLines {
			w.debugPanel.lines = w.debugPanel.lines[len(w.debugPanel.lines)-w.debugPanel.maxLines:]
		}

		// Update display immediately if visible
		if shouldUpdate {
			w.debugPanel.updateDisplayLocked()
		}
		w.debugPanel.mu.Unlock()
	}

	return len(p), nil
}
