package poe_log

import (
	"bufio"
	"fmt"
	"os"
	"poe-helper/internal/models"
	"poe-helper/internal/poe/window"
	"poe-helper/pkg/logger"
	"regexp"
	"strings"
	"time"
)

var timestampRegex = regexp.MustCompile(`^\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}:\d{2}`)

type LogWatcher struct {
	poeLogPath  string
	log         *logger.Logger
	triggers    map[string]*regexp.Regexp
	handler     func(models.TradeEntry)
	windowCheck *window.Detector
	stopChan    chan struct{}
}

func NewLogWatcher(poeLogPath string, log *logger.Logger, triggers map[string]*regexp.Regexp, handler func(models.TradeEntry)) (*LogWatcher, error) {
	log.Debug("Initializing new LogWatcher",
		"path", poeLogPath,
		"trigger_count", len(triggers))

	detector := window.NewDetector(log)
	watcher := &LogWatcher{
		poeLogPath:  poeLogPath,
		log:         log,
		triggers:    triggers,
		handler:     handler,
		windowCheck: detector,
		stopChan:    make(chan struct{}),
	}

	if err := detector.Start(); err != nil {
		log.Error("Failed to start window detector", err)
		return nil, fmt.Errorf("failed to start window detector: %w", err)
	}

	log.Debug("LogWatcher initialized successfully")
	return watcher, nil
}

func (w *LogWatcher) Watch() error {
	w.log.Info("Starting log watch routine", "path", w.poeLogPath)

	file, err := os.Open(w.poeLogPath)
	if err != nil {
		w.log.Error("Failed to open log file", err)
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Get initial file size
	stat, _ := file.Stat()
	initialSize := stat.Size()
	w.log.Info("Initial file size", "size", initialSize)

	// Instead of seeking to end immediately, we'll keep track of where we need to read from
	var offset int64 = initialSize
	lastSize := initialSize

	// Increase scanner buffer size to handle long lines
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)

	for {
		select {
		case <-w.stopChan:
			return nil
		default:
			// Check current file size
			stat, err := file.Stat()
			if err != nil {
				w.log.Error("Failed to stat file", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			currentSize := stat.Size()

			// Handle file truncation
			if currentSize < lastSize {
				w.log.Info("File was truncated, resetting",
					"old_size", lastSize,
					"new_size", currentSize)
				offset = 0
				lastSize = 0
			}

			// If there's new content
			if currentSize > offset {
				// Seek to where we left off
				if _, err := file.Seek(offset, 0); err != nil {
					w.log.Error("Failed to seek file", err)
					time.Sleep(500 * time.Millisecond)
					continue
				}

				// Create new scanner for this read
				scanner := bufio.NewScanner(file)
				scanner.Buffer(buf, maxScanTokenSize)

				// Read all new lines
				for scanner.Scan() {
					line := scanner.Text()
					w.log.Debug("Read new line",
						"content", line[:min(len(line), 100)],
						"length", len(line))

					if err := w.processLogLine(line); err != nil {
						w.log.Debug("Failed to process log line",
							"error", err)
					}
				}

				if err := scanner.Err(); err != nil {
					w.log.Error("Scanner error", err)
					time.Sleep(500 * time.Millisecond)
					continue
				}

				// Update our offset
				offset = currentSize
				lastSize = currentSize
			}

			time.Sleep(500 * time.Millisecond)
		}
	}
}

func (w *LogWatcher) processLogLine(line string) error {
	// Parse timestamp
	timestamp, err := w.parseTimestamp(line)
	if err != nil {
		w.log.Debug("Failed to parse timestamp",
			"line", line,
			"error", err)
		return nil
	}

	// Check if log line should be processed based on window and session state
	if !w.windowCheck.CheckLogLineValidity(timestamp, line) {
		return nil
	}

	// Process trade messages
	for triggerName, trigger := range w.triggers {
		matches := trigger.FindStringSubmatch(line)
		if len(matches) > 1 {
			playerName := matches[1]
			w.log.Info("Triggered event",
				"trigger", triggerName,
				"player", playerName,
				"line", line,
			)

			// Create the trade entry
			entry := models.TradeEntry{
				Timestamp:   timestamp,
				TriggerType: triggerName,
				PlayerName:  playerName,
				Message:     line,
			}

			// Call the trade entry callback if provided
			if w.handler != nil {
				w.handler(entry)
			}
		}
	}

	return nil
}

func (w *LogWatcher) parseTimestamp(line string) (time.Time, error) {
	parts := strings.SplitN(line, " ", 4)
	if len(parts) < 4 {
		return time.Time{}, fmt.Errorf("insufficient parts in line")
	}

	timestampStr := fmt.Sprintf("%s %s", parts[0], parts[1])
	return time.Parse("2006/01/02 15:04:05", timestampStr)
}

func (w *LogWatcher) Stop() {
	w.log.Info("Stopping log watcher")
	close(w.stopChan)
	// w.windowCheck.Stop()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
