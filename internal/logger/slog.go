package logger

import (
	"fmt"
	"log/slog"
	"os"
)

// SlogLogger implements Logger interface using the standard library log/slog.
type SlogLogger struct {
	sLogger *slog.Logger
}

// NewSlogLogger creates a new Logger instance backed by standard slog.
func NewSlogLogger(isProd bool) Logger {
	var handler slog.Handler
	if isProd {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	return &SlogLogger{sLogger: slog.New(handler)}
}

// Sugar returns the logger itself.
func (sl *SlogLogger) Sugar() Logger {
	return sl
}

// With creates a child logger with additional structured context fields.
func (sl *SlogLogger) With(keysAndValues ...any) Logger {
	return &SlogLogger{sLogger: sl.sLogger.With(keysAndValues...)}
}

// Debug logs a message at DebugLevel with structured key-value pairs.
func (sl *SlogLogger) Debug(msg string, keysAndValues ...any) {
	sl.sLogger.Debug(msg, keysAndValues...)
}

// Info logs a message at InfoLevel with structured key-value pairs.
func (sl *SlogLogger) Info(msg string, keysAndValues ...any) {
	sl.sLogger.Info(msg, keysAndValues...)
}

// Warn logs a message at WarnLevel with structured key-value pairs.
func (sl *SlogLogger) Warn(msg string, keysAndValues ...any) {
	sl.sLogger.Warn(msg, keysAndValues...)
}

// Error logs a message at ErrorLevel with structured key-value pairs.
func (sl *SlogLogger) Error(msg string, keysAndValues ...any) {
	sl.sLogger.Error(msg, keysAndValues...)
}

// Debugf logs a message at DebugLevel using fmt.Sprintf formatting.
func (sl *SlogLogger) Debugf(template string, args ...any) {
	sl.sLogger.Debug(fmt.Sprintf(template, args...))
}

// Infof logs a message at InfoLevel using fmt.Sprintf formatting.
func (sl *SlogLogger) Infof(template string, args ...any) {
	sl.sLogger.Info(fmt.Sprintf(template, args...))
}

// Warnf logs a message at WarnLevel using fmt.Sprintf formatting.
func (sl *SlogLogger) Warnf(template string, args ...any) {
	sl.sLogger.Warn(fmt.Sprintf(template, args...))
}

// Errorf logs a message at ErrorLevel using fmt.Sprintf formatting.
func (sl *SlogLogger) Errorf(template string, args ...any) {
	sl.sLogger.Error(fmt.Sprintf(template, args...))
}

// Sync is a no-op for slog, as it does not require flushing.
func (sl *SlogLogger) Sync() error {
	return nil
}
