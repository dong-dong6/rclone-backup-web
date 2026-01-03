package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelStrings = []string{
	"DEBUG",
	"INFO",
	"WARN",
	"ERROR",
	"FATAL",
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	EventCode   string                 `json:"event_code,omitempty"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Caller      string                 `json:"caller,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
}

// Logger is the main logger struct
type Logger struct {
	mu          sync.Mutex
	level       LogLevel
	output      io.Writer
	errorOutput io.Writer
	locale      string
	messages    map[string]map[string]string // locale -> event_code -> message template
	includeCallerInfo bool
}

// New creates a new logger instance
func New(level LogLevel, output io.Writer) *Logger {
	return &Logger{
		level:       level,
		output:      output,
		errorOutput: os.Stderr,
		locale:      "zh",
		messages:    make(map[string]map[string]string),
		includeCallerInfo: true,
	}
}

// LoadMessages loads localized message templates
func (l *Logger) LoadMessages(locale string, messages map[string]string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.messages[locale] == nil {
		l.messages[locale] = make(map[string]string)
	}
	
	for code, msg := range messages {
		l.messages[locale][code] = msg
	}
}

// LoadMessagesFromFile loads messages from a JSON file
func (l *Logger) LoadMessagesFromFile(locale string, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read message file: %w", err)
	}
	
	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("failed to parse message file: %w", err)
	}
	
	l.LoadMessages(locale, messages)
	return nil
}

// SetLocale sets the current locale for message formatting
func (l *Logger) SetLocale(locale string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.locale = locale
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// formatMessage formats a message template with parameters
func (l *Logger) formatMessage(eventCode string, details map[string]interface{}) string {
	l.mu.Lock()
	messages, exists := l.messages[l.locale]
	l.mu.Unlock()
	
	if !exists || messages[eventCode] == "" {
		// Fallback to English if available
		if enMessages, ok := l.messages["en"]; ok {
			if template, ok := enMessages[eventCode]; ok {
				return l.interpolate(template, details)
			}
		}
		// Return event code if no template found
		return eventCode
	}
	
	template := messages[eventCode]
	return l.interpolate(template, details)
}

// interpolate replaces placeholders in template with actual values
func (l *Logger) interpolate(template string, details map[string]interface{}) string {
	result := template
	for key, value := range details {
		placeholder := fmt.Sprintf("{%s}", key)
		result = replaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// replaceAll replaces all occurrences of old with new in s
func replaceAll(s, old, new string) string {
	// Simple implementation, in production use strings.ReplaceAll
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			break
		}
		s = s[:idx] + new + s[idx+len(old):]
	}
	return s
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// log is the internal logging function
func (l *Logger) log(level LogLevel, eventCode string, message string, details map[string]interface{}) {
	if level < l.level {
		return
	}
	
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     levelStrings[level],
		EventCode: eventCode,
		Message:   message,
		Details:   details,
	}
	
	// Add caller information
	if l.includeCallerInfo && level >= WARN {
		if pc, file, line, ok := runtime.Caller(2); ok {
			entry.Caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
			
			// Add stack trace for errors
			if level >= ERROR {
				fn := runtime.FuncForPC(pc)
				if fn != nil {
					entry.StackTrace = fn.Name()
				}
			}
		}
	}
	
	// Format and write the log entry
	l.mu.Lock()
	defer l.mu.Unlock()
	
	output := l.output
	if level >= ERROR && l.errorOutput != nil {
		output = l.errorOutput
	}
	
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	encoder.Encode(entry)
}

// Event logs an event with event code
func (l *Logger) Event(level LogLevel, eventCode string, details map[string]interface{}) {
	message := l.formatMessage(eventCode, details)
	l.log(level, eventCode, message, details)
}

// Debug logs a debug message
func (l *Logger) Debug(eventCode string, details map[string]interface{}) {
	l.Event(DEBUG, eventCode, details)
}

// Info logs an info message
func (l *Logger) Info(eventCode string, details map[string]interface{}) {
	l.Event(INFO, eventCode, details)
}

// Warn logs a warning message
func (l *Logger) Warn(eventCode string, details map[string]interface{}) {
	l.Event(WARN, eventCode, details)
}

// Error logs an error message
func (l *Logger) Error(eventCode string, details map[string]interface{}) {
	l.Event(ERROR, eventCode, details)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(eventCode string, details map[string]interface{}) {
	l.Event(FATAL, eventCode, details)
	os.Exit(1)
}

// WithDetails creates a child logger with preset details
func (l *Logger) WithDetails(details map[string]interface{}) *ContextLogger {
	return &ContextLogger{
		logger:  l,
		details: details,
	}
}

// ContextLogger is a logger with preset context details
type ContextLogger struct {
	logger  *Logger
	details map[string]interface{}
}

// mergeDetails merges context details with provided details
func (cl *ContextLogger) mergeDetails(details map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	
	// Copy context details
	for k, v := range cl.details {
		merged[k] = v
	}
	
	// Override with provided details
	for k, v := range details {
		merged[k] = v
	}
	
	return merged
}

// Debug logs a debug message with context
func (cl *ContextLogger) Debug(eventCode string, details map[string]interface{}) {
	cl.logger.Debug(eventCode, cl.mergeDetails(details))
}

// Info logs an info message with context
func (cl *ContextLogger) Info(eventCode string, details map[string]interface{}) {
	cl.logger.Info(eventCode, cl.mergeDetails(details))
}

// Warn logs a warning message with context
func (cl *ContextLogger) Warn(eventCode string, details map[string]interface{}) {
	cl.logger.Warn(eventCode, cl.mergeDetails(details))
}

// Error logs an error message with context
func (cl *ContextLogger) Error(eventCode string, details map[string]interface{}) {
	cl.logger.Error(eventCode, cl.mergeDetails(details))
}