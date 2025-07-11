package logger

import (
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type LogCapture struct {
	mu           sync.RWMutex
	logs         []LogEntry
	suppressLogs bool
	maxLogs      int
	originalOut  io.Writer
}

type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

var (
	capture *LogCapture
	once    sync.Once
)

func init() {
	capture = &LogCapture{
		logs:         make([]LogEntry, 0),
		suppressLogs: false,
		maxLogs:      1000,
		originalOut:  os.Stdout,
	}
}

// CustomWriter intercepts log output and stores it
type CustomWriter struct {
	originalWriter io.Writer
}

func (cw *CustomWriter) Write(p []byte) (n int, err error) {
	logMessage := string(p)

	// Parse the log message to extract level and clean message
	level, message := parseLogMessage(logMessage)

	// Always store the log entry
	capture.addLog(level, message)

	// Only write to original output if not suppressed
	if !capture.suppressLogs {
		return cw.originalWriter.Write(p)
	}
	// Return the length as if we wrote it (to satisfy io.Writer interface)
	return len(p), nil
}

// parseLogMessage attempts to parse the log message and extract level and clean message
func parseLogMessage(logMessage string) (string, string) {
	// Remove timestamp prefix that Go's standard logger adds
	// Pattern: "2006/01/02 15:04:05 message"
	timestampPattern := regexp.MustCompile(`^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	cleanMessage := timestampPattern.ReplaceAllString(logMessage, "")

	// Trim whitespace and newlines
	cleanMessage = strings.TrimSpace(cleanMessage)

	// Determine log level based on content
	level := "INFO"
	lowerMessage := strings.ToLower(cleanMessage)

	if strings.Contains(lowerMessage, "error") ||
		strings.Contains(lowerMessage, "failed") ||
		strings.Contains(lowerMessage, "panic") {
		level = "ERROR"
	} else if strings.Contains(lowerMessage, "warning") ||
		strings.Contains(lowerMessage, "warn") {
		level = "WARN"
	} else if strings.Contains(lowerMessage, "debug") {
		level = "DEBUG"
	}

	return level, cleanMessage
}

func (l *LogCapture) addLog(level, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Skip empty messages
	if message == "" {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	l.logs = append(l.logs, entry)

	// Keep only the last maxLogs entries
	if len(l.logs) > l.maxLogs {
		l.logs = l.logs[1:]
	}
}

// SetupLogCapture replaces the standard log output with our custom writer
func SetupLogCapture() {
	customWriter := &CustomWriter{originalWriter: capture.originalOut}
	log.SetOutput(customWriter)
	// Remove timestamp from Go's standard logger since we add our own
	log.SetFlags(0)
}

// SetSuppress controls whether logs are displayed in console
func SetSuppress(suppress bool) {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	capture.suppressLogs = suppress
}

// GetLogs returns a copy of all captured logs
func GetLogs() []LogEntry {
	capture.mu.RLock()
	defer capture.mu.RUnlock()
	logs := make([]LogEntry, len(capture.logs))
	copy(logs, capture.logs)
	return logs
}

// ClearLogs removes all stored logs
func ClearLogs() {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	capture.logs = make([]LogEntry, 0)
}

// RestoreOriginalOutput restores the original log output (for cleanup)
func RestoreOriginalOutput() {
	log.SetOutput(capture.originalOut)
	log.SetFlags(log.LstdFlags) // Restore default flags
}

func GetLogCount() int {
	capture.mu.RLock()
	defer capture.mu.RUnlock()
	return len(capture.logs)
}
