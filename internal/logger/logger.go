// internal/logger/logger.go
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Log levels
const (
	LevelDebug    = 0
	LevelInfo     = 1
	LevelWarning  = 2
	LevelError    = 3
	LevelCritical = 4
)

// Logger provides logging functionality
type Logger struct {
	mu         sync.Mutex
	file       *os.File
	writers    []io.Writer
	logLevel   int
	timeFormat string
}

// NewLogger creates a new logger
func NewLogger(filePath string) (*Logger, error) {
	logger := &Logger{
		logLevel:   LevelInfo,
		timeFormat: "2006-01-02 15:04:05",
		writers:    []io.Writer{os.Stdout}, // Default to stdout
	}

	// If a file path is provided, open the log file
	if filePath != "" {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %v", err)
		}

		// Open log file
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %v", err)
		}

		logger.file = file
		logger.writers = append(logger.writers, file)
	}

	return logger, nil
}

// Close closes the logger and any associated resources
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logLevel = level
}

// log logs a message with the given level
func (l *Logger) log(level int, component, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if we should log this level
	if level < l.logLevel {
		return
	}

	// Format timestamp
	timestamp := time.Now().Format(l.timeFormat)

	// Format level
	var levelStr string
	switch level {
	case LevelDebug:
		levelStr = "DEBUG"
	case LevelInfo:
		levelStr = "INFO"
	case LevelWarning:
		levelStr = "WARNING"
	case LevelError:
		levelStr = "ERROR"
	case LevelCritical:
		levelStr = "CRITICAL"
	default:
		levelStr = "UNKNOWN"
	}

	// Format message
	msg := fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp, levelStr, component, message)

	// Write to all writers
	for _, writer := range l.writers {
		writer.Write([]byte(msg))
	}
}

// Debug logs a debug message
func (l *Logger) Debug(component, message string) {
	l.log(LevelDebug, component, message)
}

// Info logs an info message
func (l *Logger) Info(component, message string) {
	l.log(LevelInfo, component, message)
}

// Warning logs a warning message
func (l *Logger) Warning(component, message string) {
	l.log(LevelWarning, component, message)
}

// Error logs an error message
func (l *Logger) Error(component, message string) {
	l.log(LevelError, component, message)
}

// Critical logs a critical message
func (l *Logger) Critical(component, message string) {
	l.log(LevelCritical, component, message)
}
