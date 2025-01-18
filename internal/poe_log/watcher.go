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
	"poe-helper/pkg/logger"
)

type LogWatcher struct {
	logPath          string
	log              *logger.Logger
	lastSize         int64
	windowFoundTime  time.Time
	sessionStartTime time.Time
	foundSession     bool
	stopChan         chan struct{}
	mu               sync.RWMutex

	// Callbacks
	onTradeEntry     func(models.TradeEntry)
	compiledTriggers map[string]*regexp.Regexp `json:"-"`
}

// NewLogWatcher creates a new log watcher
func NewLogWatcher(
	logPath string,
	log *logger.Logger,
	windowFoundTime time.Time,
	compiledTriggers map[string]*regexp.Regexp,
	onTradeEntry func(models.TradeEntry),
) *LogWatcher {
	return &LogWatcher{
		logPath:          logPath,
		log:              log,
		windowFoundTime:  windowFoundTime,
		foundSession:     false,
		sessionStartTime: time.Time{},
		stopChan:         make(chan struct{}),
		compiledTriggers: compiledTriggers,
		onTradeEntry:     onTradeEntry,
	}
}

// Watch starts monitoring the log file
func (l *LogWatcher) Watch() {
	l.log.Info("Starting log watcher",
		"path", l.logPath,
		"window_found_time", l.windowFoundTime,
		"triggers_count", len(l.compiledTriggers),
	)

	// Get initial file size
	stat, err := os.Stat(l.logPath)
	if err != nil {
		l.log.Error("Failed to stat log file", err)
		return
	}

	lastSize := stat.Size()
	var lastError error
	var lastErrorTime time.Time

	for {
		select {
		case <-l.stopChan:
			l.log.Info("Stopping log watcher")
			return
		default:
			stat, err := os.Stat(l.logPath)
			if err != nil {
				if lastError == nil || err.Error() != lastError.Error() ||
					time.Since(lastErrorTime) > time.Minute {
					l.log.Error("Failed to stat log file", err)
					lastError = err
					lastErrorTime = time.Now()
				}
				time.Sleep(5 * time.Second)
				continue
			}

			// If file is truncated or rotated, reset to beginning
			if stat.Size() < lastSize {
				l.log.Info("Log file was truncated, resetting position",
					"old_size", lastSize,
					"new_size", stat.Size(),
				)
				lastSize = 0
			}

			if stat.Size() > lastSize {
				file, err := os.Open(l.logPath)
				if err != nil {
					l.log.Error("Failed to open log file", err)
					time.Sleep(5 * time.Second)
					continue
				}

				// Seek to last read position
				if lastSize > 0 {
					_, err = file.Seek(lastSize, 0)
					if err != nil {
						l.log.Error("Failed to seek in log file", err)
						file.Close()
						continue
					}
				}

				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := scanner.Text()

					// Parse timestamp
					parts := strings.SplitN(line, " ", 4)
					if len(parts) < 4 {
						l.log.Debug("Skipping line - insufficient parts", "line", line)
						continue
					}

					timestampStr := fmt.Sprintf("%s %s", parts[0], parts[1])
					timestamp, err := time.Parse("2006/01/02 15:04:05", timestampStr)
					if err != nil {
						l.log.Debug("Failed to parse timestamp",
							"line", line,
							"timestamp_str", timestampStr,
							"error", err,
						)
						continue
					}

					// Filtering logic
					l.mu.RLock()
					windowTime := l.windowFoundTime
					foundSession := l.foundSession
					l.mu.RUnlock()

					// Check if line is after window found time
					if timestamp.After(windowTime) {
						// Check for session start or trade message
						if !foundSession {
							if strings.Contains(line, "[STARTUP] Loading Start") {
								l.mu.Lock()
								l.sessionStartTime = timestamp
								l.foundSession = true
								l.mu.Unlock()

								l.log.Info("Found new POE session start",
									"time", timestamp,
								)
							}
						}

						// Process trade messages
						l.processLogLine(line, timestamp)
					} else {
						l.log.Debug("Skipping line before window time",
							"line_time", timestamp,
							"window_time", windowTime,
						)
					}
				}

				file.Close()
				lastSize = stat.Size()
			}

			time.Sleep(100 * time.Millisecond)
		}
	}
}

// shouldProcessLine determines if a line should be processed
func (l *LogWatcher) shouldProcessLine(line string, timestamp time.Time) bool {
	l.mu.RLock()
	windowTime := l.windowFoundTime
	foundSession := l.foundSession
	l.mu.RUnlock()

	l.log.Debug("Should process line check",
		"line", line,
		"timestamp", timestamp,
		"window_time", windowTime,
		"found_session", foundSession,
	)

	// If we haven't found the session start yet, look for it
	if !foundSession {
		if strings.Contains(line, "[STARTUP] Loading Start") && timestamp.After(windowTime) {
			l.mu.Lock()
			l.sessionStartTime = timestamp
			l.foundSession = true
			l.mu.Unlock()

			l.log.Info("Found new POE session start",
				"time", timestamp,
				"window_found_time", windowTime,
			)
			return true
		}
		return false
	}

	// Log the session start time for reference
	l.log.Debug("Checking line against session start",
		"line", line,
		"timestamp", timestamp,
		"session_start_time", l.sessionStartTime,
	)

	// Only process lines after session start
	return timestamp.After(l.sessionStartTime) || timestamp.Equal(l.sessionStartTime)
}

// processLogLine handles a single log line
func (l *LogWatcher) processLogLine(line string, timestamp time.Time) {
	for triggerName, trigger := range l.compiledTriggers {
		matches := trigger.FindStringSubmatch(line)
		if len(matches) > 1 {
			playerName := matches[1]
			l.log.Info("Triggered event",
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
			if l.onTradeEntry != nil {
				l.onTradeEntry(entry)
			}
		}
	}
}

// Stop signals the watcher to stop
func (l *LogWatcher) Stop() {
	l.log.Info("Stopping log watcher")
	close(l.stopChan)
}
