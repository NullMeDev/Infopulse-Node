// internal/logger/logger.go
package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/NullMeDev/Infopulse-Node/internal/models"
)

// LogLevel represents severity level for logging
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	CRITICAL
)

// Logger handles logging to console and file
type Logger struct {
	fileLogger  *log.Logger
	consoleLogger *log.Logger
	logFilePath string
	logFile     *os.File
}

// NewLogger creates a new logger instance
func NewLogger(logFilePath string) (*Logger, error) {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// Open log file
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Create loggers
	fileLogger := log.New(logFile, "", log.Ldate|log.Ltime)
	consoleLogger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	return &Logger{
		fileLogger:   fileLogger,
		consoleLogger: consoleLogger,
		logFilePath:  logFilePath,
		logFile:      logFile,
	}, nil
}

// LogString logs a message with the specified level and source
func (l *Logger) LogString(level LogLevel, source, message string) {
	levelStr := getLevelString(level)
	timestamp := time.Now()

	// Format log message
	logMessage := fmt.Sprintf("[%s] [%s] %s", levelStr, source, message)

	// Log to console and file
	l.consoleLogger.Println(logMessage)
	l.fileLogger.Println(logMessage)
}

// LogIntel logs intelligence data
func (l *Logger) LogIntel(level LogLevel, source string, intel *models.Intelligence) {
	levelStr := getLevelString(level)
	timestamp := time.Now()

	// Format log message for intelligence data
	logMessage := fmt.Sprintf("[%s] [%s] [Category: %s] %s - %s", 
		levelStr, source, intel.Category, intel.Title, intel.URL)

	// Log to console and file
	l.consoleLogger.Println(logMessage)
	l.fileLogger.Println(logMessage)

	// TODO: Store intelligence in database for historical tracking
}

// Log different severity levels
func (l *Logger) Debug(source, message string) {
	l.LogString(DEBUG, source, message)
}

func (l *Logger) Info(source, message string) {
	l.LogString(INFO, source, message)
}

func (l *Logger) Warning(source, message string) {
	l.LogString(WARNING, source, message)
}

func (l *Logger) Error(source, message string) {
	l.LogString(ERROR, source, message)
}

func (l *Logger) Critical(source, message string) {
	l.LogString(CRITICAL, source, message)
}

// Helper function to convert log level to string
func getLevelString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	case CRITICAL:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}
