package poe_log

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"poe-helper/internal/models"
	"poe-helper/internal/poe/window"

	"poe-helper/pkg/global"
)

// Only match lines that start with a valid timestamp
var timestampRegex = regexp.MustCompile(`^\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}:\d{2}`)

type LogWatcher struct {
	handler     func(models.TradeEntry)
	windowCheck *window.Detector
	stopChan    chan struct{}
	mu          sync.Mutex
	stopped     bool
}

func NewLogWatcher(handler func(models.TradeEntry)) (*LogWatcher, error) {
	cfg, log, _ := global.GetAll()
	log.Debug("Initializing new LogWatcher",
		"path", cfg.PoeLogPath,
		"trigger_count", len(cfg.Triggers))

	detector := window.NewDetector()
	watcher := &LogWatcher{
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
	cfg, log, _ := global.GetAll()
	log.Info("Starting log watch routine", "path", cfg.PoeLogPath)

	file, err := os.Open(cfg.PoeLogPath)
	if err != nil {
		log.Error("Failed to open log file", err)
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Create done channel for cleanup signaling
	done := make(chan struct{})
	defer close(done)

	// Start the watch loop in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- w.watchLoop(file)
	}()

	// Wait for either stop signal or error
	select {
	case <-w.stopChan:
		log.Info("Received stop signal")
		return nil
	case err := <-errChan:
		return err
	}
}

func (w *LogWatcher) watchLoop(file *os.File) error {
	log := global.GetLogger()

	// Get initial file size
	stat, _ := file.Stat()
	initialSize := stat.Size()
	log.Info("Initial file size", "size", initialSize)

	// Instead of seeking to end immediately, we'll keep track of where we need to read from
	var offset = initialSize
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
				log.Error("Failed to stat file", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			currentSize := stat.Size()

			// Handle file truncation
			if currentSize < lastSize {
				log.Info("File was truncated, resetting",
					"old_size", lastSize,
					"new_size", currentSize)
				offset = 0
				lastSize = 0
			}

			// If there's new content
			if currentSize > offset {
				// Seek to where we left off
				if _, err := file.Seek(offset, 0); err != nil {
					log.Error("Failed to seek file", err)
					time.Sleep(500 * time.Millisecond)
					continue
				}

				// Create new scanner for this read
				scanner := bufio.NewScanner(file)
				scanner.Buffer(buf, maxScanTokenSize)

				// Read all new lines
				for scanner.Scan() {
					line := scanner.Text()
					log.Debug("Read new line",
						"content", line[:min(len(line), 100)],
						"length", len(line))

					if err := w.processLogLine(line); err != nil {
						log.Debug("Failed to process log line",
							"error", err)
					}
				}

				if err := scanner.Err(); err != nil {
					log.Error("Scanner error", err)
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
	cfg, log, _ := global.GetAll()

	// Check if line starts with a valid timestamp
	if !timestampRegex.MatchString(line) {
		log.Debug("Rejecting line - invalid timestamp format",
			"line", line)
		return nil
	}

	// Parse timestamp
	timestamp, err := w.parseTimestamp(line)
	if err != nil {
		log.Debug("Failed to parse timestamp",
			"line", line,
			"error", err)
		return nil
	}

	// Check if log line should be processed based on window and session state
	if !w.windowCheck.CheckLogLineValidity(timestamp, line) {
		return nil
	}

	// Process trade messages
	for triggerName, trigger := range cfg.CompiledTriggers {
		matches := trigger.FindStringSubmatch(line)
		if len(matches) > 1 {
			playerName := matches[1]
			log.Info("Triggered event",
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

func (w *LogWatcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return nil
	}

	log := global.GetLogger()
	log.Info("Stopping log watcher")
	// Signal the watch routine to stop
	close(w.stopChan)

	// Stop the window detector
	if err := w.windowCheck.Stop(); err != nil {
		log.Error("Failed to stop window detector", err)
		return fmt.Errorf("failed to stop window detector: %w", err)
	}

	w.stopped = true
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
