package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const (
	DefaultLogDir  = "~/.local/share/poe-helper/logs"
	DefaultLogFile = "debug.log"
)

type Logger struct {
	zlog    zerolog.Logger
	file    *os.File
	writers []io.Writer
	mu      sync.RWMutex
}

type Option func(*Logger) error

// WithConsole enables console logging
func WithConsole() Option {
	return func(l *Logger) error {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		l.zlog = l.zlog.Output(consoleWriter)
		return nil
	}
}

// WithLevel sets the logging level
func WithLevel(level zerolog.Level) Option {
	return func(l *Logger) error {
		l.zlog = l.zlog.Level(level)
		return nil
	}
}

// WithFile sets up file logging with an explicit path
func WithFile(path string) Option {
	return func(l *Logger) error {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		fileWriter := zerolog.ConsoleWriter{
			Out:        f,
			TimeFormat: time.RFC3339,
			NoColor:    true,
		}
		l.file = f
		l.zlog = l.zlog.Output(fileWriter)
		return nil
	}
}

// getDefaultLogPath returns the expanded default log path
func getDefaultLogPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	logDir := strings.Replace(DefaultLogDir, "~", homeDir, 1)
	return filepath.Join(logDir, DefaultLogFile), nil
}

// NewLogger creates a new logger with the given options
func NewLogger(opts ...Option) (*Logger, error) {
	logger := &Logger{
		zlog: zerolog.New(os.Stderr).With().Timestamp().Logger(),
	}

	// If no file option is provided, use the default log path
	hasFileOption := false
	for _, opt := range opts {
		if fmt.Sprintf("%p", opt) == fmt.Sprintf("%p", WithFile("")) {
			hasFileOption = true
			break
		}
	}

	if !hasFileOption {
		defaultPath, err := getDefaultLogPath()
		if err != nil {
			return nil, fmt.Errorf("failed to get default log path: %w", err)
		}
		opts = append([]Option{WithFile(defaultPath)}, opts...)
	}

	// Apply all options
	for _, opt := range opts {
		if err := opt(logger); err != nil {
			return nil, fmt.Errorf("failed to apply logger option: %w", err)
		}
	}

	return logger, nil
}

// Close closes the logger and any open files
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// addSourceContext adds file and line information to the event
func addSourceContext(e *zerolog.Event) *zerolog.Event {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		return e.Str("file", filepath.Base(file)).Int("line", line)
	}
	return e
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...interface{}) {
	event := addSourceContext(l.zlog.Debug())
	logFields(event, fields...)
	event.Msg(msg)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...interface{}) {
	event := addSourceContext(l.zlog.Info())
	logFields(event, fields...)
	event.Msg(msg)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...interface{}) {
	event := addSourceContext(l.zlog.Warn())
	logFields(event, fields...)
	event.Msg(msg)
}

// Error logs an error message
func (l *Logger) Error(msg string, err error, fields ...interface{}) {
	event := addSourceContext(l.zlog.Error())
	if err != nil {
		event = event.Err(err)
	}
	logFields(event, fields...)
	event.Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, err error, fields ...interface{}) {
	event := addSourceContext(l.zlog.Fatal())
	if err != nil {
		event = event.Err(err)
	}
	logFields(event, fields...)
	event.Msg(msg)
}

// AddWriter adds a writer to the logger
func (l *Logger) AddWriter(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writers = append(l.writers, w)
}

// logFields adds fields to the log event
func logFields(event *zerolog.Event, fields ...interface{}) {
	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			break
		}
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		value := fields[i+1]
		event.Interface(key, value)
	}
}
